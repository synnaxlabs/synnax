package schema

type Type string

type Schema struct {
	Type   Type
	Fields map[string]Field
}
