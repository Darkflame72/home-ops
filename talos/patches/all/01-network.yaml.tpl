apiVersion: v1alpha1
kind: LinkAliasConfig
name: ethSel0
selector:
  match: glob("{{ .Node.Data.mac }}", mac(link.hardware_addr))
---
apiVersion: v1alpha1
kind: LinkConfig
name: ethSel0
mtu: 1500
addresses:
  - address: {{ .Node.IP }}/24
routes:
  - gateway: 10.0.60.1
{{- if eq .Node.Role "control-plane" }}
---
apiVersion: v1alpha1
kind: Layer2VIPConfig
name: {{ .Node.Data.vip }}
link: ethSel0
{{- end }}
