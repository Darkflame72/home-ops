// alert-relay receives Alertmanager webhook notifications, hands each
// firing alert to kagent's sre-agent over A2A for diagnosis, and posts the
// result to Discord with a click-to-acknowledge link. Investigations are
// deduplicated per-alertname via a ConfigMap so a flapping/still-firing
// alert doesn't re-trigger the agent until a human acknowledges the prior
// diagnosis.
//
// Go port of the original Python/FastAPI service (removed in 42f025f along
// with the rest of the kagent stack); behavior is intentionally unchanged.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	saDir         = "/var/run/secrets/kubernetes.io/serviceaccount"
	k8sAPIBase    = "https://kubernetes.default.svc"
	statusPending = "pending"
	statusAcked   = "acknowledged"
	claimRetries  = 5
	ackRetries    = 5
)

var (
	kagentBaseURL      string
	kagentNamespace    string
	kagentAgent        string
	kagentTimeout      time.Duration
	discordWebhookURL  string
	ackBaseURL         string
	stateNamespace     string
	stateConfigMapName string
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required environment variable %s", key)
	}
	return v
}

func main() {
	kagentBaseURL = requireEnv("KAGENT_BASE_URL")
	kagentNamespace = getenv("KAGENT_NAMESPACE", "kagent")
	kagentAgent = getenv("KAGENT_AGENT", "sre-agent")
	timeoutSecs, err := strconv.ParseFloat(getenv("KAGENT_TIMEOUT_SECONDS", "180"), 64)
	if err != nil {
		log.Fatalf("invalid KAGENT_TIMEOUT_SECONDS: %v", err)
	}
	kagentTimeout = time.Duration(timeoutSecs * float64(time.Second))
	discordWebhookURL = requireEnv("DISCORD_WEBHOOK_URL")
	ackBaseURL = requireEnv("ACK_BASE_URL")
	stateNamespace = getenv("STATE_NAMESPACE", "kagent")
	stateConfigMapName = getenv("STATE_CONFIGMAP_NAME", "alert-relay-state")

	mux := http.NewServeMux()
	mux.HandleFunc("POST /alertmanager", handleAlertmanager)
	mux.HandleFunc("GET /ack/{token}", handleAck)
	mux.HandleFunc("GET /healthz", handleHealthz)

	log.Print("alert-relay listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

// --- Kubernetes API client ---------------------------------------------

var k8sHTTPClient = newK8sHTTPClient()

func newK8sHTTPClient() *http.Client {
	caCert, err := os.ReadFile(saDir + "/ca.crt")
	if err != nil {
		log.Fatalf("failed to read service account CA cert: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		log.Fatal("failed to parse service account CA cert")
	}
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}
}

// k8sRequest issues an authenticated request against the in-cluster API
// server. The token is re-read per request since projected service account
// tokens rotate.
func k8sRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	token, err := os.ReadFile(saDir + "/token")
	if err != nil {
		return nil, fmt.Errorf("read service account token: %w", err)
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, k8sAPIBase+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+string(bytes.TrimSpace(token)))
	req.Header.Set("Content-Type", "application/json")
	return k8sHTTPClient.Do(req)
}

// --- Dedup/ack state -----------------------------------------------------

type stateEntry struct {
	Status         string `json:"status"`
	AckToken       string `json:"ack_token"`
	TriggeredAt    string `json:"triggered_at"`
	AcknowledgedAt string `json:"acknowledged_at,omitempty"`
}

type configMap struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   configMapMetadata `json:"metadata"`
	Data       map[string]string `json:"data"`
}

type configMapMetadata struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace,omitempty"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
}

// loadState fetches and decodes the dedup ConfigMap. Returns an empty map
// and an empty resourceVersion if it doesn't exist yet.
func loadState(ctx context.Context) (map[string]stateEntry, string, error) {
	path := fmt.Sprintf("/api/v1/namespaces/%s/configmaps/%s", stateNamespace, stateConfigMapName)
	resp, err := k8sRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return map[string]stateEntry{}, "", nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("get configmap: %s: %s", resp.Status, body)
	}

	var cm configMap
	if err := json.Unmarshal(body, &cm); err != nil {
		return nil, "", err
	}
	raw, ok := cm.Data["state.json"]
	if !ok || raw == "" {
		raw = "{}"
	}
	var state map[string]stateEntry
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, "", err
	}
	return state, cm.Metadata.ResourceVersion, nil
}

// saveState writes the dedup ConfigMap. Returns ok=false on a resourceVersion
// conflict (409), in which case the caller should reload and retry.
func saveState(ctx context.Context, state map[string]stateEntry, resourceVersion string) (bool, error) {
	raw, err := json.Marshal(state)
	if err != nil {
		return false, err
	}

	var cm configMap
	cm.APIVersion = "v1"
	cm.Kind = "ConfigMap"
	cm.Data = map[string]string{"state.json": string(raw)}

	var method, path string
	if resourceVersion == "" {
		cm.Metadata = configMapMetadata{Name: stateConfigMapName, Namespace: stateNamespace}
		method = http.MethodPost
		path = fmt.Sprintf("/api/v1/namespaces/%s/configmaps", stateNamespace)
	} else {
		cm.Metadata = configMapMetadata{Name: stateConfigMapName, ResourceVersion: resourceVersion}
		method = http.MethodPut
		path = fmt.Sprintf("/api/v1/namespaces/%s/configmaps/%s", stateNamespace, stateConfigMapName)
	}

	body, err := json.Marshal(cm)
	if err != nil {
		return false, err
	}
	resp, err := k8sRequest(ctx, method, path, body)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return false, nil
	}
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("%s configmap: %s: %s", method, resp.Status, respBody)
	}
	return true, nil
}

// claimInvestigation claims the dedup lock for alertname. Returns an ack
// token if the caller should investigate, or "" if a prior diagnosis for
// this alertname is still pending review.
func claimInvestigation(ctx context.Context, alertname string) (string, error) {
	for i := 0; i < claimRetries; i++ {
		state, resourceVersion, err := loadState(ctx)
		if err != nil {
			return "", err
		}
		if entry, ok := state[alertname]; ok && entry.Status == statusPending {
			return "", nil
		}
		token := newUUID()
		state[alertname] = stateEntry{
			Status:      statusPending,
			AckToken:    token,
			TriggeredAt: time.Now().UTC().Format(time.RFC3339),
		}
		ok, err := saveState(ctx, state, resourceVersion)
		if err != nil {
			return "", err
		}
		if ok {
			return token, nil
		}
	}
	return "", fmt.Errorf("failed to claim dedup lock for %s after retries", alertname)
}

// acknowledge marks the alertname owning this ack token as acknowledged.
// Returns "" if the token doesn't match any pending entry.
func acknowledge(ctx context.Context, token string) (string, error) {
	for i := 0; i < ackRetries; i++ {
		state, resourceVersion, err := loadState(ctx)
		if err != nil {
			return "", err
		}
		var match string
		for name, entry := range state {
			if entry.AckToken == token {
				match = name
				break
			}
		}
		if match == "" {
			return "", nil
		}
		entry := state[match]
		entry.Status = statusAcked
		entry.AcknowledgedAt = time.Now().UTC().Format(time.RFC3339)
		state[match] = entry

		ok, err := saveState(ctx, state, resourceVersion)
		if err != nil {
			return "", err
		}
		if ok {
			return match, nil
		}
	}
	return "", fmt.Errorf("failed to record acknowledgement after retries")
}

// --- kagent A2A call -------------------------------------------------

type alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

type a2aPart struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type a2aArtifact struct {
	Parts []a2aPart `json:"parts"`
}

type a2aMessage struct {
	Role  string    `json:"role"`
	Parts []a2aPart `json:"parts"`
}

type a2aResult struct {
	Artifacts []a2aArtifact `json:"artifacts"`
	History   []a2aMessage  `json:"history"`
}

type a2aResponse struct {
	Result *a2aResult      `json:"result"`
	Error  json.RawMessage `json:"error"`
}

func buildPrompt(a alert) string {
	return fmt.Sprintf(
		"Alertmanager alert firing: %s\nSeverity: %s\nStarts at: %s\nLabels: %v\nAnnotations: %v\nSource: %s",
		orUnknown(a.Labels["alertname"]),
		orUnknown(a.Labels["severity"]),
		orUnknown(a.StartsAt),
		a.Labels,
		a.Annotations,
		orUnknown(a.GeneratorURL),
	)
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}

func extractText(result *a2aResult) string {
	if result == nil {
		return ""
	}
	if len(result.Artifacts) > 0 {
		last := result.Artifacts[len(result.Artifacts)-1]
		if text := joinTextParts(last.Parts); text != "" {
			return text
		}
	}
	for i := len(result.History) - 1; i >= 0; i-- {
		msg := result.History[i]
		if msg.Role != "agent" {
			continue
		}
		if text := joinTextParts(msg.Parts); text != "" {
			return text
		}
	}
	return ""
}

func joinTextParts(parts []a2aPart) string {
	var texts []string
	for _, p := range parts {
		if p.Kind == "text" && p.Text != "" {
			texts = append(texts, p.Text)
		}
	}
	out := ""
	for i, t := range texts {
		if i > 0 {
			out += "\n"
		}
		out += t
	}
	return out
}

func invokeAgent(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/api/a2a/%s/%s/", kagentBaseURL, kagentNamespace, kagentAgent)
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      newUUID(),
		"method":  "message/send",
		"params": map[string]any{
			"message": map[string]any{
				"role":      "user",
				"kind":      "message",
				"messageId": newUUID(),
				"parts":     []map[string]any{{"kind": "text", "text": prompt}},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, kagentTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: kagentTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("kagent A2A call: %s: %s", resp.Status, respBody)
	}

	var parsed a2aResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Error) > 0 {
		return "", fmt.Errorf("kagent error: %s", parsed.Error)
	}
	if text := extractText(parsed.Result); text != "" {
		return text, nil
	}
	return "(sre-agent returned no text response)", nil
}

// --- Discord ---------------------------------------------------------

func postDiscord(ctx context.Context, a alert, diagnosis, ackToken string) error {
	alertname := orUnknown(a.Labels["alertname"])
	severity := orUnknown(a.Labels["severity"])
	namespace := a.Labels["namespace"]
	if namespace == "" {
		namespace = "-"
	}
	ackURL := ackBaseURL + "/ack/" + ackToken

	desc := diagnosis
	if len(desc) > 3800 {
		desc = desc[:3800]
	}
	desc += fmt.Sprintf("\n\n[Mark as reviewed](%s)", ackURL)

	color := 0xE74C3C
	if severity == "warning" {
		color = 0xE67E22
	}

	embed := map[string]any{
		"title":       fmt.Sprintf("Kagent diagnosis: %s", alertname),
		"description": desc,
		"color":       color,
		"fields": []map[string]any{
			{"name": "severity", "value": severity, "inline": true},
			{"name": "namespace", "value": namespace, "inline": true},
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	payload := map[string]any{"embeds": []map[string]any{embed}}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discordWebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("post discord: %s: %s", resp.Status, respBody)
	}
	return nil
}

// --- Alert processing --------------------------------------------------

func processAlert(a alert) {
	alertname := orUnknown(a.Labels["alertname"])
	ctx := context.Background()

	ackToken, err := claimInvestigation(ctx, alertname)
	if err != nil {
		log.Printf("failed to claim dedup lock for alert %s: %v", alertname, err)
		return
	}
	if ackToken == "" {
		log.Printf("skipping %s: a prior diagnosis is still awaiting review", alertname)
		return
	}

	diagnosis, err := invokeAgent(ctx, buildPrompt(a))
	if err != nil {
		log.Printf("sre-agent investigation failed for alert %s: %v", alertname, err)
		return
	}

	if err := postDiscord(ctx, a, diagnosis, ackToken); err != nil {
		log.Printf("failed to post diagnosis to discord for alert %s: %v", alertname, err)
	}
}

// --- HTTP handlers -----------------------------------------------------

func handleAlertmanager(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Alerts []alert `json:"alerts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	var firing []alert
	for _, a := range payload.Alerts {
		if a.Status == "firing" {
			firing = append(firing, a)
		}
	}
	for _, a := range firing {
		go processAlert(a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"accepted": len(firing)})
}

func handleAck(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	alertname, err := acknowledge(r.Context(), token)
	w.Header().Set("Content-Type", "text/html")
	if err != nil {
		log.Printf("failed to record acknowledgement for token %s: %v", token, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "<h1>Failed to record acknowledgement, try again.</h1>")
		return
	}
	if alertname == "" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "<h1>Unknown or already-acknowledged link.</h1>")
		return
	}
	fmt.Fprintf(w, "<h1>Acknowledged: %s</h1><p>kagent will investigate again next time this alert fires.</p>", alertname)
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// --- UUID ---------------------------------------------------------------

// newUUID generates a random RFC 4122 v4 UUID without pulling in an
// external dependency.
func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read only fails if the OS RNG is unavailable, which
		// would make the rest of this service non-functional anyway.
		log.Fatalf("failed to generate uuid: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
