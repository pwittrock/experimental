package markdown

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"tektoncd.dev/experimental/pkg/cligithub"
	"tektoncd.dev/experimental/pkg/clik8s"
	"tektoncd.dev/experimental/pkg/deprecated/objects"
)

type Markdowner struct {
	Path clik8s.ResourceConfigPath
	Name cligithub.Name
}

type Body struct {
	Path    string
	Objects []*objects.Object
	Status  string
	Name    string
	Header  string
}

func (m *Markdowner) GetMarkdown(objs []*objects.Object) (string, error) {
	status := "Complete"
	for _, o := range objs {
		if o.Done != true {
			status = "In Progress"
		}
	}

	b := &bytes.Buffer{}
	if err := issueTemplate.Execute(b, Body{
		Path:    string(m.Path),
		Objects: objs,
		Status:  status,
		Name:    string(m.Name),
		Header:  fmt.Sprintf("**%s**", time.Now().Format("Mon Jan _2 15:04:05 2006")),
	}); err != nil {
		return "", err
	}
	return b.String(), nil
}

const commentTemplateBody = "# {{ .Name }} " + `

## {{ .Header }} *{{ .Status }}*
---

{{ range $obj := .Objects }}
- [{{ if $obj.Done}}x{{else}} {{end}}] {{ $obj.Display }}
{{ if $obj.Status }}  - ` + "**rollout:** `{{ $obj.Status}}`" + `
{{range $h := $obj.History }}    - {{ $h }}
{{ end }}
{{ end -}}
{{ end }}
`

var issueTemplate = template.Must(template.New("comment").Parse(commentTemplateBody))
