package main

const entryTmpl = `
entity "{{ .Name }}" {
{{- if .Comment.Valid }}
  {{ .Comment.String }}
  ..
{{- end }}
{{- range .Columns }}
  {{- if .IsPrimaryKey }}
  + {{ .Name }} [PK]{{- if .Comment.Valid }} : {{ .Comment.String }}{{- end }}
  {{- end }}
{{- end }}
  --
{{- range .Columns }}
  {{- if not .IsPrimaryKey }}
  {{ .Name }} {{- if .Comment.Valid }} : {{ .Comment.String }}{{- end }}
  {{- end }}
{{- end }}
}
`

const relationTmpl = `
{{ .SourceTableName }} "0..N" -- "1" {{ .TargetTableName }}
`

const tableTmpl = `@startuml
!ifndef ERD_INCL
!include ../erd.iuml
!endif
table({{ .Name }}) {
{{- range .Columns }}
  {{- if .IsPrimaryKey }}
  pk({{ .Name }}): {{ .DataType }} {{- if .NotNull }} NN{{- end }}
  {{- else }}
  {{ .Name }}{{- if .DefVal.Valid }} = {{ .DefVal.String }} {{- end }}: {{ .DataType }} {{- if .NotNull }} NN{{- end }} {{- if .IsUnique }} UN{{- end }}
  {{- end }}
{{- end }}
}
@enduml`

const rstTableTmpl = `
.. _tab-sql-{{ .Schema }}_{{ .Name }}:

{{ .Name }}
^^^^^^^

{{ if .Comment.Valid }}{{ .Comment.String }} {{- else }}TODO_ADD_COMMENT{{- end }}

.. tabularcolumns:: |p{3cm}|p{3cm}|p{8cm}|

.. csv-table:: {{ .Name }}
   :header: column,type,description
{{ range .Columns }}
   "{{ .Name }}", "{{ .DataType }}", "{{- if .Comment.Valid }}{{ .Comment.String }} {{- else }}TODO_ADD_COMMENT{{- end }}"
{{- end }}
`
