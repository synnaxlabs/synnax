package schema

type Entity struct {
	Schema *Schema                `json:"schema" msgpack:"schema"`
	Name   string                 `json:"name" msgpack:"name"`
	Data   map[string]interface{} `json:"data" msgpack:"data"`
}

func Get[V Value](d Entity, k string) (v V, ok bool) {
	rv, ok := d.Data[k]
	if !ok {
		return v, false
	}
	v, ok = rv.(V)
	if !ok {
		panic("[Schema] - invalid field type")
	}
	return v, true
}

func Set[V Value](D Entity, k string, v V) {
	f, ok := D.Schema.Fields[k]
	if !ok {
		panic("[Schema] - field not found")
	}
	if !f.Type.AssertValue(v) {
		panic("[Schema] - invalid field type")
	}
	D.Data[k] = v
}

func NewEntity(schema *Schema, name string) Entity {
	return Entity{
		Schema: schema,
		Name:   name,
		Data:   map[string]interface{}{},
	}
}
