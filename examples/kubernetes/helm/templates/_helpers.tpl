{{/*
Expand the name of the chart.
*/}}
{{- define "mercator-jupiter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "mercator-jupiter.fullname" -}}
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
{{- define "mercator-jupiter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "mercator-jupiter.labels" -}}
helm.sh/chart: {{ include "mercator-jupiter.chart" . }}
{{ include "mercator-jupiter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "mercator-jupiter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mercator-jupiter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "mercator-jupiter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "mercator-jupiter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the secret to use
*/}}
{{- define "mercator-jupiter.secretName" -}}
{{- if .Values.secrets.existingSecret }}
{{- .Values.secrets.existingSecret }}
{{- else }}
{{- include "mercator-jupiter.fullname" . }}-secrets
{{- end }}
{{- end }}

{{/*
Create the name of the config ConfigMap to use
*/}}
{{- define "mercator-jupiter.configMapName" -}}
{{- include "mercator-jupiter.fullname" . }}-config
{{- end }}

{{/*
Create the name of the policies ConfigMap to use
*/}}
{{- define "mercator-jupiter.policiesConfigMapName" -}}
{{- if .Values.policies.existingConfigMap }}
{{- .Values.policies.existingConfigMap }}
{{- else }}
{{- include "mercator-jupiter.fullname" . }}-policies
{{- end }}
{{- end }}

{{/*
Get the image tag
*/}}
{{- define "mercator-jupiter.imageTag" -}}
{{- .Values.image.tag | default .Chart.AppVersion }}
{{- end }}
