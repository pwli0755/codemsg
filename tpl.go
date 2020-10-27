package main

const suffix = "_msg_gen"

// tpl temp
const tpl = `
// Code generated by github.com/pwli0755/codemsg DO NOT EDIT
// Source: {{.pkg}}/{{.source}}
// {{.file}} is a generated file.

package {{.pkg}}

// messages get msg from const comment
var messages = map[{{.constType}}]string{
	{{range $key, $value := .comments}}
	{{$key}}: "{{$value}}",{{end}}
}

// GetMsg get code msg
func GetMsg(code {{.constType}}) string {
	if msg, ok := messages[code]; ok {
		return msg
	}
	return "UNKNOWN ERROR"
}
`