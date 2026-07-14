import asyncio
import json
import logging
import os
import uuid
from datetime import UTC, datetime

import httpx
from fastapi import FastAPI, Request
from fastapi.responses import HTMLResponse

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("alert-relay")

app = FastAPI()

KAGENT_BASE_URL = os.environ["KAGENT_BASE_URL"]
KAGENT_NAMESPACE = os.environ.get("KAGENT_NAMESPACE", "kagent")
KAGENT_AGENT = os.environ.get("KAGENT_AGENT", "sre-agent")
KAGENT_TIMEOUT_SECONDS = float(os.environ.get("KAGENT_TIMEOUT_SECONDS", "180"))
DISCORD_WEBHOOK_URL = os.environ["DISCORD_WEBHOOK_URL"]
ACK_BASE_URL = os.environ["ACK_BASE_URL"]

STATE_NAMESPACE = os.environ.get("STATE_NAMESPACE", "kagent")
STATE_CONFIGMAP_NAME = os.environ.get("STATE_CONFIGMAP_NAME", "alert-relay-state")
SA_DIR = "/var/run/secrets/kubernetes.io/serviceaccount"
K8S_API_BASE = "https://kubernetes.default.svc"

STATUS_PENDING = "pending"
STATUS_ACKNOWLEDGED = "acknowledged"


def _k8s_client() -> httpx.AsyncClient:
    with open(f"{SA_DIR}/token") as f:
        token = f.read().strip()
    return httpx.AsyncClient(
        base_url=K8S_API_BASE,
        headers={"Authorization": f"Bearer {token}"},
        verify=f"{SA_DIR}/ca.crt",
        timeout=10,
    )


async def _load_state(client: httpx.AsyncClient) -> tuple[dict, str | None]:
    resp = await client.get(f"/api/v1/namespaces/{STATE_NAMESPACE}/configmaps/{STATE_CONFIGMAP_NAME}")
    if resp.status_code == 404:
        return {}, None
    resp.raise_for_status()
    body = resp.json()
    raw = (body.get("data") or {}).get("state.json", "{}")
    return json.loads(raw), body["metadata"]["resourceVersion"]


async def _save_state(client: httpx.AsyncClient, data: dict, resource_version: str | None) -> bool:
    """Returns True on success, False on a resourceVersion conflict (caller should retry)."""
    body = {
        "apiVersion": "v1",
        "kind": "ConfigMap",
        "metadata": {"name": STATE_CONFIGMAP_NAME},
        "data": {"state.json": json.dumps(data)},
    }
    if resource_version is None:
        body["metadata"] = {"name": STATE_CONFIGMAP_NAME, "namespace": STATE_NAMESPACE}
        resp = await client.post(f"/api/v1/namespaces/{STATE_NAMESPACE}/configmaps", json=body)
        if resp.status_code == 409:
            return False
        resp.raise_for_status()
        return True

    body["metadata"]["resourceVersion"] = resource_version
    resp = await client.put(
        f"/api/v1/namespaces/{STATE_NAMESPACE}/configmaps/{STATE_CONFIGMAP_NAME}",
        json=body,
    )
    if resp.status_code == 409:
        return False
    resp.raise_for_status()
    return True


async def _claim_investigation(client: httpx.AsyncClient, alertname: str) -> str | None:
    """Claims the dedup lock for alertname. Returns an ack token if this call should
    investigate, or None if a prior diagnosis for this alertname is still pending review."""
    for _ in range(5):
        data, resource_version = await _load_state(client)
        entry = data.get(alertname)
        if entry and entry.get("status") == STATUS_PENDING:
            return None
        token = str(uuid.uuid4())
        data[alertname] = {
            "status": STATUS_PENDING,
            "ack_token": token,
            "triggered_at": datetime.now(UTC).isoformat(),
        }
        if await _save_state(client, data, resource_version):
            return token
    raise RuntimeError(f"failed to claim dedup lock for {alertname} after retries")


async def _acknowledge(client: httpx.AsyncClient, token: str) -> str | None:
    """Marks the alertname owning this ack token as acknowledged. Returns the alertname, or
    None if the token doesn't match any pending entry."""
    for _ in range(5):
        data, resource_version = await _load_state(client)
        match = next(
            (name for name, entry in data.items() if entry.get("ack_token") == token),
            None,
        )
        if match is None:
            return None
        data[match]["status"] = STATUS_ACKNOWLEDGED
        data[match]["acknowledged_at"] = datetime.now(UTC).isoformat()
        if await _save_state(client, data, resource_version):
            return match
    raise RuntimeError("failed to record acknowledgement after retries")


def _extract_text(result: dict) -> str | None:
    artifacts = result.get("artifacts") or []
    if artifacts:
        parts = artifacts[-1].get("parts", [])
        texts = [p.get("text", "") for p in parts if p.get("kind") == "text" and p.get("text")]
        if texts:
            return "\n".join(texts)

    for msg in reversed(result.get("history") or []):
        if msg.get("role") != "agent":
            continue
        texts = [p.get("text", "") for p in msg.get("parts", []) if p.get("kind") == "text" and p.get("text")]
        if texts:
            return "\n".join(texts)

    return None


def _build_prompt(alert: dict) -> str:
    labels = alert.get("labels", {})
    annotations = alert.get("annotations", {})
    return "\n".join(
        [
            f"Alertmanager alert firing: {labels.get('alertname', 'unknown')}",
            f"Severity: {labels.get('severity', 'unknown')}",
            f"Starts at: {alert.get('startsAt', 'unknown')}",
            f"Labels: {labels}",
            f"Annotations: {annotations}",
            f"Source: {alert.get('generatorURL', 'unknown')}",
        ]
    )


async def _invoke_agent(client: httpx.AsyncClient, prompt: str) -> str:
    url = f"{KAGENT_BASE_URL}/api/a2a/{KAGENT_NAMESPACE}/{KAGENT_AGENT}/"
    payload = {
        "jsonrpc": "2.0",
        "id": str(uuid.uuid4()),
        "method": "message/send",
        "params": {
            "message": {
                "role": "user",
                "kind": "message",
                "messageId": str(uuid.uuid4()),
                "parts": [{"kind": "text", "text": prompt}],
            }
        },
    }
    resp = await client.post(url, json=payload, timeout=KAGENT_TIMEOUT_SECONDS)
    resp.raise_for_status()
    body = resp.json()
    if "error" in body:
        raise RuntimeError(f"kagent error: {body['error']}")
    return _extract_text(body.get("result", {})) or "(sre-agent returned no text response)"


async def _post_discord(client: httpx.AsyncClient, alert: dict, diagnosis: str, ack_token: str) -> None:
    labels = alert.get("labels", {})
    ack_url = f"{ACK_BASE_URL}/ack/{ack_token}"
    embed = {
        "title": f"Kagent diagnosis: {labels.get('alertname', 'unknown')}",
        "description": f"{diagnosis[:3800]}\n\n[Mark as reviewed]({ack_url})",
        "color": 0xE67E22 if labels.get("severity") == "warning" else 0xE74C3C,
        "fields": [
            {"name": "severity", "value": labels.get("severity", "unknown"), "inline": True},
            {"name": "namespace", "value": labels.get("namespace", "-"), "inline": True},
        ],
        "timestamp": datetime.now(UTC).isoformat(),
    }
    resp = await client.post(DISCORD_WEBHOOK_URL, json={"embeds": [embed]}, timeout=10)
    resp.raise_for_status()


async def _process_alert(alert: dict) -> None:
    alertname = alert.get("labels", {}).get("alertname", "unknown")
    async with _k8s_client() as k8s_client:
        try:
            ack_token = await _claim_investigation(k8s_client, alertname)
        except Exception:
            logger.exception("failed to claim dedup lock for alert %s", alertname)
            return

    if ack_token is None:
        logger.info("skipping %s: a prior diagnosis is still awaiting review", alertname)
        return

    async with httpx.AsyncClient() as client:
        try:
            diagnosis = await _invoke_agent(client, _build_prompt(alert))
        except Exception:
            logger.exception("sre-agent investigation failed for alert %s", alertname)
            return

        try:
            await _post_discord(client, alert, diagnosis, ack_token)
        except Exception:
            logger.exception("failed to post diagnosis to discord for alert %s", alertname)


@app.post("/alertmanager")
async def alertmanager_webhook(request: Request) -> dict:
    payload = await request.json()
    firing = [a for a in payload.get("alerts", []) if a.get("status") == "firing"]
    for alert in firing:
        asyncio.create_task(_process_alert(alert))
    return {"accepted": len(firing)}


@app.get("/ack/{token}", response_class=HTMLResponse)
async def acknowledge(token: str) -> HTMLResponse:
    async with _k8s_client() as client:
        try:
            alertname = await _acknowledge(client, token)
        except Exception:
            logger.exception("failed to record acknowledgement for token %s", token)
            return HTMLResponse("<h1>Failed to record acknowledgement, try again.</h1>", status_code=500)

    if alertname is None:
        return HTMLResponse("<h1>Unknown or already-acknowledged link.</h1>", status_code=404)

    return HTMLResponse(
        f"<h1>Acknowledged: {alertname}</h1><p>kagent will investigate again next time this alert fires.</p>"
    )


@app.get("/healthz")
async def healthz() -> dict:
    return {"status": "ok"}
