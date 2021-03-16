{{/*
Expand the name of the chart.
*/}}
{{- define "estafette-ci-api.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "estafette-ci-api.fullname" -}}
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
{{- define "estafette-ci-api.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "estafette-ci-api.labels" -}}
helm.sh/chart: {{ include "estafette-ci-api.chart" . }}
{{ include "estafette-ci-api.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- range $key, $value := .Values.extraLabels }}
{{ $key }}: {{ $value }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "estafette-ci-api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "estafette-ci-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "estafette-ci-api.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "estafette-ci-api.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the namespace for build/release jobs
*/}}
{{- define "estafette-ci-api.jobNamespace" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.jobNamespaceOverride }}
{{- else }}
{{- .Release.Namespace }}-jobs
{{- end }}
{{- end }}
