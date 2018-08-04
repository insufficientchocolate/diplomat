package diplomat

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
)

type TranslationPair struct {
	Key        string
	Translated string
}

type MessengerConfig interface {
	GetType() string
	GetName() string
	GetContent() LocaleTranslations
	GetLocale() string
	GetFragmentName() string
}

type BasicMessengerConfig struct {
	messengerType string
	name          string
	content       LocaleTranslations
	locale        string
	fragmentName  string
}

func (c BasicMessengerConfig) GetType() string {
	return c.messengerType
}

func (c BasicMessengerConfig) GetName() string {
	return c.name
}

func (c BasicMessengerConfig) GetContent() LocaleTranslations {
	return c.content
}

func (c BasicMessengerConfig) GetLocale() string {
	return c.locale
}

func (c BasicMessengerConfig) GetFragmentName() string {
	return c.fragmentName
}

// type Messenger interface {
// 	GetFolder() string
// 	Send(config MessengerConfig) error
// }

var JsMessengerTemplate *template.Template

func init() {
	var err error
	JsMessengerTemplate, err = template.New("jsMessengerTemplate").Parse(`// DO NOT EDIT. generated by diplomat (https://github.com/MinecraftXwinP/diplomat).
export default {
    {{- range $k,$v := . }}
    {{$k}}: "{{$v}}",
{{ end -}}
}
`)
	if err != nil {
		log.Fatal(err)
	}
}

func NewJsModuleMessenger(config MessengerConfig) *jsModuleMessenger {
	return &jsModuleMessenger{config}
}

type jsModuleMessenger struct {
	config MessengerConfig
}

func (j jsModuleMessenger) GetFolder() string {
	return "js"
}

func (j jsModuleMessenger) Send(writer io.Writer) error {
	return JsMessengerTemplate.Execute(writer, j.config.GetContent().Translations)
}

func (j jsModuleMessenger) getFileName() (string, error) {
	var buffer bytes.Buffer
	t, err := template.New("js").Parse(j.config.GetName())
	if err != nil {
		return "", err
	}
	err = t.Execute(
		&buffer,
		struct {
			Locale       string
			FragmentName string
		}{
			j.config.GetLocale(),
			j.config.GetFragmentName(),
		},
	)
	if err != nil {
		return "", nil
	}
	return string(buffer.Bytes()), nil
}

func JsModuleMessengerHandler(fragmentName, locale, name string, content LocaleTranslations, path string) {
	messenger := NewJsModuleMessenger(BasicMessengerConfig{
		content:       content,
		messengerType: "js",
		name:          name,
		fragmentName:  fragmentName,
		locale:        content.Locale,
	})
	filename, err := messenger.getFileName()
	if err != nil {
		log.Println(err)
	}
	f, err := os.Create(filepath.Join(path, filename))
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	err = messenger.Send(f)
	if err != nil {
		log.Println(err)
	}
}
