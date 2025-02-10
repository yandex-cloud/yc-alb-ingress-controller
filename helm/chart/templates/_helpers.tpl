{{- define "yc-alb-ingress-controller.fullname" -}}
	{{- default .Chart.Name .Values.nameOverride -}}
{{- end -}}

{{- define "yc-alb-ingress-controller.serviceAccountName" -}}
	{{- if .Values.serviceAccount.create -}}
    	{{ default (include "yc-alb-ingress-controller.fullname" .) .Values.serviceAccount.name }}
	{{- else -}}
    	{{ default "default" .Values.serviceAccount.name }}
	{{- end -}}
{{- end -}}


{{- define "validateRegionFunc" -}}
    {{- if or (or (eq . "a") (eq . "vla")) (eq . "ru-central1-a")}}
    {{- print "ru-central1-a" }}
    {{- else if or (or (eq . "b") (eq . "sas")) (eq . "ru-central1-b")}}
    {{- print "ru-central1-b" }}
    {{- else if or (or (eq . "c") (eq . "myt")) (eq . "ru-central1-c")}}
    {{- print "ru-central1-c" }}
    {{- else if eq . "il1-a" }}
    {{- print "il1-a" }}
    {{- else if eq . "" }}
    {{- print "" }}
    {{- else }}
    {{ fail ".Values.region is not valid"}}
    {{- end }}
{{- end -}}

{{- define "validateClusterID" -}}
    {{- if regexMatch "^[0-9a-z]{20}$" . }}
    {{- . }}
    {{- else if eq . "" }}
    {{- print "" }}
    {{- else }}
    {{ fail "ClusterID must match regexp ^[0-9a-z]{20}"}}
    {{- end}}
{{- end -}}