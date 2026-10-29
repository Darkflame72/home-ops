#!/bin/sh
# Mirrors every PrometheusRule in the cluster into Mimir's ruler, one rule
# namespace per PrometheusRule (k8s/<namespace>/<name>), with one ruler rule
# group per spec.groups entry. Replaces the mimir.rules.kubernetes component
# Alloy used to run, through the same ruler HTTP API. Only keys under the
# k8s/ prefix are managed here; anything else in the ruler is left alone.
set -eu

RULER="${RULER_URL:-http://mimir-ruler.observability.svc.cluster.local:8080}"
PREFIX=k8s
work=$(mktemp -d)

encode() {
  jq -rn --arg v "$1" '$v | @uri'
}

# Rule namespaces the ruler holds under our prefix. The ruler returns them as
# top-level YAML keys.
ruler_namespaces() {
  curl -fsS "$RULER/prometheus/config/v1/rules" | grep -E '^[^[:space:]].*:$' | sed 's/:$//' | grep "^$PREFIX/" || true
}

# Group names in one ruler namespace. Each group is a top-level list item under
# the namespace key. Empty output when the namespace does not exist.
ruler_groups() {
  curl -s "$RULER/prometheus/config/v1/rules/$(encode "$1")" | sed -n 's/^    - name: //p'
}

delete_group() {
  curl -fsS -X DELETE "$RULER/prometheus/config/v1/rules/$(encode "$1")/$(encode "$2")" > /dev/null
  echo "removed group $2 from $1"
}

kubectl get prometheusrules -A -o json > "$work/rules.json"
jq -c '.items[]' "$work/rules.json" > "$work/items"

: > "$work/desired"
while read -r item; do
  ns=$(printf '%s' "$item" | jq -r '.metadata.namespace')
  name=$(printf '%s' "$item" | jq -r '.metadata.name')
  key="$PREFIX/$ns/$name"
  printf '%s\n' "$key" >> "$work/desired"

  # The ruler upserts one rule group per POST. JSON is a subset of YAML, so
  # each group is sent as-is.
  printf '%s' "$item" | jq -c '.spec.groups[]' > "$work/groups"
  while read -r group; do
    printf '%s' "$group" > "$work/body"
    code=$(curl -s -o "$work/resp" -w '%{http_code}' -X POST \
      -H 'Content-Type: application/yaml' --data-binary @"$work/body" \
      "$RULER/prometheus/config/v1/rules/$(encode "$key")")
    case "$code" in
      2*) echo "synced $key/$(printf '%s' "$group" | jq -r '.name')" ;;
      *)
        echo "failed to sync $key (HTTP $code):" >&2
        cat "$work/resp" >&2
        exit 1
        ;;
    esac
  done < "$work/groups"

  # Drop groups that were removed from this PrometheusRule.
  printf '%s' "$item" | jq -r '.spec.groups[].name' > "$work/want"
  ruler_groups "$key" > "$work/have"
  grep -vxFf "$work/want" "$work/have" > "$work/stale" || true
  while read -r stale; do
    delete_group "$key" "$stale"
  done < "$work/stale"
done < "$work/items"

# Drop whole rule namespaces whose PrometheusRule no longer exists.
ruler_namespaces > "$work/keys"
grep -vxFf "$work/desired" "$work/keys" > "$work/stale-ns" || true
while read -r key; do
  ruler_groups "$key" > "$work/have"
  while read -r group; do
    delete_group "$key" "$group"
  done < "$work/have"
done < "$work/stale-ns"

rm -rf "$work"
