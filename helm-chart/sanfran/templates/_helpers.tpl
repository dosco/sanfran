{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "sanfran.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "sanfran.fullname" -}}
{{- $name := base .Template.Name | trimSuffix ".yaml" -}}
{{- printf "%s-sf-%s" .Release.Name $name | trunc 63 | trimSuffix "-" | lower -}}
{{- end -}}
