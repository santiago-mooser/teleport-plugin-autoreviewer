{{/*
Expand the name of the chart.
*/}}
{{- define "teleport-plugin-request-autoreviewer.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "teleport-plugin-request-autoreviewer.fullname" -}}
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
{{- define "teleport-plugin-request-autoreviewer.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "teleport-plugin-request-autoreviewer.labels" -}}
helm.sh/chart: {{ include "teleport-plugin-request-autoreviewer.chart" . }}
{{ include "teleport-plugin-request-autoreviewer.selectorLabels" . }}
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
{{- define "teleport-plugin-request-autoreviewer.selectorLabels" -}}
app.kubernetes.io/name: {{ include "teleport-plugin-request-autoreviewer.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "teleport-plugin-request-autoreviewer.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "teleport-plugin-request-autoreviewer.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the container image reference
*/}}
{{- define "teleport-plugin-request-autoreviewer.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}

{{/*
Common annotations
*/}}
{{- define "teleport-plugin-request-autoreviewer.annotations" -}}
{{- with .Values.commonAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Pod security context
*/}}
{{- define "teleport-plugin-request-autoreviewer.podSecurityContext" -}}
{{- toYaml .Values.podSecurityContext }}
{{- end }}

{{/*
Container security context
*/}}
{{- define "teleport-plugin-request-autoreviewer.securityContext" -}}
{{- toYaml .Values.securityContext }}
{{- end }}

{{/*
Generate config.yaml content
*/}}
{{- define "teleport-plugin-request-autoreviewer.config" -}}
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
{{- define "teleport-plugin-request-autoreviewer.podTemplateAnnotations" -}}
checksum/config: {{ include "teleport-plugin-request-autoreviewer.config" . | sha256sum }}
{{- if .Values.teleport.identityFile }}
checksum/secret: {{ .Values.teleport.identityFile | sha256sum }}
{{- end }}
{{- with .Values.podAnnotations }}
{{ toYaml . }}
{{- end }}
{{- with (include "teleport-plugin-request-autoreviewer.annotations" .) }}
{{ . }}
{{- end }}
{{- end }}

{{/*
Generate pod template labels
*/}}
{{- define "teleport-plugin-request-autoreviewer.podTemplateLabels" -}}
{{ include "teleport-plugin-request-autoreviewer.selectorLabels" . }}
{{- with .Values.podLabels }}
{{ toYaml . }}
{{- end }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
tbot helpers
*/}}
{{- define "teleport-plugin-request-autoreviewer.tbot.fullname" -}}
{{- printf "%s-tbot" (include "teleport-plugin-request-autoreviewer.fullname" .) }}
{{- end }}

{{/*
tbot selector labels
*/}}
{{- define "teleport-plugin-request-autoreviewer.tbot.selectorLabels" -}}
app.kubernetes.io/name: {{ include "teleport-plugin-request-autoreviewer.name" . }}-tbot
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
tbot labels
*/}}
{{- define "teleport-plugin-request-autoreviewer.tbot.labels" -}}
helm.sh/chart: {{ include "teleport-plugin-request-autoreviewer.chart" . }}
{{ include "teleport-plugin-request-autoreviewer.tbot.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
tbot image reference
*/}}
{{- define "teleport-plugin-request-autoreviewer.tbot.image" -}}
{{- printf "%s:%s" .Values.tbot.image.repository .Values.tbot.image.tag }}
{{- end }}

{{/*
tbot service account name
*/}}
{{- define "teleport-plugin-request-autoreviewer.tbot.serviceAccountName" -}}
{{- printf "%s-tbot" (include "teleport-plugin-request-autoreviewer.serviceAccountName" .) }}
{{- end }}

{{/*
Get teleport cluster address for tbot
*/}}
{{- define "teleport-plugin-request-autoreviewer.tbot.addr" -}}
{{- if .Values.tbot.authentication.addr }}
{{- .Values.tbot.authentication.addr }}
{{- else }}
{{- .Values.teleport.addr }}
{{- end }}
{{- end }}

{{/*
Get tbot output secret name
*/}}
{{- define "teleport-plugin-request-autoreviewer.tbot.outputSecretName" -}}
{{- if .Values.tbot.output.secretName }}
{{- .Values.tbot.output.secretName }}
{{- else }}
{{- printf "%s-teleport-identity" .Release.Name }}
{{- end }}
{{- end }}

{{/*
Get tbot join method
*/}}
{{- define "teleport-plugin-request-autoreviewer.tbot.joinMethod" -}}
{{- .Values.tbot.authentication.method | default "kubernetes" }}
{{- end }}

{{/*
Validate tbot configuration
*/}}
{{- define "teleport-plugin-request-autoreviewer.tbot.validate" -}}
{{- if .Values.tbot.enabled }}
  {{- if not .Values.tbot.authentication.token }}
    {{- fail "tbot.authentication.token is required when tbot.enabled is true" }}
  {{- end }}
  {{- if and (eq (include "teleport-plugin-request-autoreviewer.tbot.joinMethod" .) "kubernetes") .Values.tbot.authentication.clusterName }}
    {{- if not .Values.tbot.authentication.clusterName }}
      {{- fail "tbot.authentication.clusterName is required when using kubernetes join method for cross-cluster authentication" }}
    {{- end }}
  {{- end }}
{{- end }}
{{- end }}

{{/*
Generate tbot configuration
*/}}
{{- define "teleport-plugin-request-autoreviewer.tbot.config" -}}
{{- include "teleport-plugin-request-autoreviewer.tbot.validate" . }}
version: v2
{{- if eq (include "teleport-plugin-request-autoreviewer.tbot.joinMethod" .) "kubernetes" }}
proxy_server: {{ include "teleport-plugin-request-autoreviewer.tbot.addr" . }}
{{- else }}
auth_server: {{ include "teleport-plugin-request-autoreviewer.tbot.addr" . }}
{{- end }}
onboarding:
  join_method: {{ include "teleport-plugin-request-autoreviewer.tbot.joinMethod" . }}
  token: {{ .Values.tbot.authentication.token | quote }}
  {{- if and (eq (include "teleport-plugin-request-autoreviewer.tbot.joinMethod" .) "kubernetes") .Values.tbot.authentication.clusterName }}
  kubernetes_cluster: {{ .Values.tbot.authentication.clusterName | quote }}
  {{- end }}
storage:
  type: kubernetes_secret
  name: {{ include "teleport-plugin-request-autoreviewer.tbot.fullname" . }}-storage
outputs:
- type: identity
  destination:
    type: kubernetes_secret
    name: {{ include "teleport-plugin-request-autoreviewer.tbot.outputSecretName" . }}
  {{- if .Values.tbot.output.renewInterval }}
  renew:
    interval: {{ .Values.tbot.output.renewInterval }}
  {{- end }}
  {{- if .Values.tbot.output.certificateLifetime }}
  identity:
    ttl: {{ .Values.tbot.output.certificateLifetime }}
  {{- end }}
{{- if .Values.tbot.config.additional }}
{{- toYaml .Values.tbot.config.additional | nindent 0 }}
{{- end }}
{{- end }}

{{/*
Check if tbot should use in-cluster authentication
*/}}
{{- define "teleport-plugin-request-autoreviewer.tbot.useInCluster" -}}
{{- if eq (include "teleport-plugin-request-autoreviewer.tbot.joinMethod" .) "kubernetes" }}
{{- true }}
{{- else }}
{{- false }}
{{- end }}
{{- end }}

{{/*
Get the identity secret name (either tbot-managed or manual)
*/}}
{{- define "teleport-plugin-request-autoreviewer.identitySecretName" -}}
{{- if .Values.tbot.enabled }}
{{- include "teleport-plugin-request-autoreviewer.tbot.outputSecretName" . }}
{{- else }}
{{- printf "%s-identity" (include "teleport-plugin-request-autoreviewer.fullname" .) }}
{{- end }}
{{- end }}
