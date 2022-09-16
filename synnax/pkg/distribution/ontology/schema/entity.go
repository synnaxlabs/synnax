package schema

type Entity struct {
	schema *Schema
	data   map[string]interface{}
}

func Get[V Value](d Entity, k string) (v V, ok bool) {
	rv, ok := d.data[k]
	if !ok {
		return v, false
	}
	v, ok = rv.(V)
	if !ok {
		panic("[schema] - invalid field type")
	}
	return v, true
}

func Set[V Value](D Entity, k string, v V) {
	f, ok := D.schema.Fields[k]
	if !ok {
		panic("[schema] - field not found")
	}
	if !f.Type.AssertValue(v) {
		panic("[schema] - invalid field type")
	}
	D.data[k] = v
}

func NewEntity(schema *Schema) Entity {
	return Entity{
		schema: schema,
		data:   map[string]interface{}{},
	}
}
