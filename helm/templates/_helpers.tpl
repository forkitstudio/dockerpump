{{- define "app.module-affinity" }}
{{- if . }}
affinity:
{{ toYaml . | indent 2 }}
{{- end }}
{{- end }}
