package diplomat

import (
	"bytes"
	"fmt"
)

func ExampleJsModuleMessengerSend() {
	config := BasicMessengerConfig{
		messengerType: "js",
		name:          "{{.FragmentName}}.{{.Locale}}.js",
		content: LocaleTranslations{
			Locale: "zh-TW",
			Translations: map[string]string{
				"admin_user": "管理員",
			},
		},
		locale:       "zh-TW",
		fragmentName: "admin",
	}
	messenger := NewJsModuleMessenger(config)
	var buffer bytes.Buffer
	messenger.Send(&buffer)
	fmt.Println(string(buffer.Bytes()))
	// Output:
	// // DO NOT EDIT. generated by diplomat (https://github.com/MinecraftXwinP/diplomat).
	// export default {
	//     admin_user: "管理員",
	// }
}
