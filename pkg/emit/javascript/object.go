package javascript

import (
	"github.com/tony84727/diplomat/pkg/data"
	"github.com/tony84727/diplomat/pkg/emit"
	"github.com/tony84727/diplomat/pkg/emit/types"
)

const javascriptTemplate = `// DO NOT EDIT. generated by diplomat (https://github.com/tony84727/diplomat).
export default {
{{ range .Pairs -}}
    "{{ JoinKeys .Key "." }}": "{{ .Text }}",
{{- end }}
}`

type ObjectEmitter struct {
	*emit.TemplateEmitter
}

func (o ObjectEmitter) Emit(translation data.Translation) ([]byte, error) {
	return o.TemplateEmitter.Emit(translation)
}

func NewObjectEmitter() types.Emitter {
	templateEmitter, err := emit.NewTemplateEmitter(javascriptTemplate)
	if err != nil {
		panic(err)
	}
	return &ObjectEmitter{
		templateEmitter,
	}
}

func init() {
	emit.GlobalRegistry.Registry("js-object", types.FactoryWrapper{Constructor: NewObjectEmitter})
}
