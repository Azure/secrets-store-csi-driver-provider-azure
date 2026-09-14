{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "sscdpa.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "sscdpa.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Common labels for helm resources
*/}}
{{- define "sscdpa.common.labels" -}}
app.kubernetes.io/instance: "{{ .Release.Name }}"
app.kubernetes.io/managed-by: "{{ .Release.Service }}"
app.kubernetes.io/version: "{{ .Chart.AppVersion }}"
helm.sh/chart: "{{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}"
{{- end -}}

{{/*
Standard labels for helm resources
*/}}
{{- define "sscdpa.labels" -}}
labels:
{{ include "sscdpa.common.labels" . | indent 2 }}
  app.kubernetes.io/name: "{{ template "sscdpa.name" . }}"
  app: {{ template "sscdpa.name" . }}
{{- end -}}

{{- define "sscdpa.psp.fullname" -}}
{{- printf "%s-psp" (include "sscdpa.fullname" .) -}}
{{- end }}

{{/*
Render the custom environment file and the content used for its pod checksum.
The provider loads its default environment only at startup, so changing this
content must roll the pods even when the mounted ConfigMap is updated.
*/}}
{{- define "sscdpa.cloudEnvironment" -}}
{{- $cloudName := required "cloud.name is required when cloud.environment is configured" .Values.cloud.name -}}
{{- if not (kindIs "string" $cloudName) -}}
{{- fail "cloud.name must be a string" -}}
{{- end -}}
{{- range $key := list "name" "activeDirectoryEndpoint" "keyVaultEndpoint" "keyVaultDNSSuffix" -}}
{{- $value := required (printf "cloud.environment.%s is required when cloud.environment is configured" $key) (index $.Values.cloud.environment $key) -}}
{{- if not (kindIs "string" $value) -}}
{{- fail (printf "cloud.environment.%s must be a string" $key) -}}
{{- end -}}
{{- end -}}
{{- .Values.cloud.environment | mustToPrettyJson -}}
{{- end -}}

{{/*
Arc specific templates
*/}}

{{- define "sscdpa.arc.labels" -}}
{{ include "sscdpa.common.labels" . }}
app.kubernetes.io/name: "arc-{{ template "sscdpa.fullname" . }}"
{{- end -}}
