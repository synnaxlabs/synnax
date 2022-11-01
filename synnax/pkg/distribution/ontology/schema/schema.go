package schema

type Type string

type Schema struct {
	Type   Type             `json:"type" msgpack:"type"`
	Fields map[string]Field `json:"fields" msgpack:"fields"`
}
