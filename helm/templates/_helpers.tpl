{{/*
Expand the name of the chart.
*/}}
{{- define "teleport-autoreviewer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "teleport-autoreviewer.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "teleport-autoreviewer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "teleport-autoreviewer.labels" -}}
helm.sh/chart: {{ include "teleport-autoreviewer.chart" . }}
{{ include "teleport-autoreviewer.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "teleport-autoreviewer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "teleport-autoreviewer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "teleport-autoreviewer.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "teleport-autoreviewer.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the container image reference
*/}}
{{- define "teleport-autoreviewer.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Common annotations
*/}}
{{- define "teleport-autoreviewer.annotations" -}}
{{- with .Values.commonAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Pod security context
*/}}
{{- define "teleport-autoreviewer.podSecurityContext" -}}
{{- toYaml .Values.podSecurityContext }}
{{- end }}

{{/*
Container security context
*/}}
{{- define "teleport-autoreviewer.securityContext" -}}
{{- toYaml .Values.securityContext }}
{{- end }}

{{/*
Generate config.yaml content
*/}}
{{- define "teleport-autoreviewer.config" -}}
teleport:
  addr: {{ .Values.teleport.addr | quote }}
  identity: "/etc/teleport/identity"
  reviewer: {{ .Values.teleport.reviewer | quote }}
  identity_refresh_interval: {{ .Values.teleport.identityRefreshInterval | quote }}

server:
  health_port: {{ .Values.server.healthPort }}
  health_path: {{ .Values.server.healthPath | quote }}

rejection:
  default_message: {{ .Values.rejection.defaultMessage | quote }}
  rules:
{{- if .Values.rejection.rules }}
{{ toYaml .Values.rejection.rules | indent 4 }}
{{- else }}
    []
{{- end }}
{{- end }}

{{/*
Generate pod template annotations
*/}}
{{- define "teleport-autoreviewer.podTemplateAnnotations" -}}
checksum/config: {{ include "teleport-autoreviewer.config" . | sha256sum }}
{{- if .Values.teleport.identityFile }}
checksum/secret: {{ .Values.teleport.identityFile | sha256sum }}
{{- end }}
{{- with .Values.podAnnotations }}
{{ toYaml . }}
{{- end }}
{{- with (include "teleport-autoreviewer.annotations" .) }}
{{ . }}
{{- end }}
{{- end }}

{{/*
Generate pod template labels
*/}}
{{- define "teleport-autoreviewer.podTemplateLabels" -}}
{{ include "teleport-autoreviewer.selectorLabels" . }}
{{- with .Values.podLabels }}
{{ toYaml . }}
{{- end }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}
