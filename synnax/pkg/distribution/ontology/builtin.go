package ontology

import "github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"

const BuiltIn Type = "builtin"

var Root = ID{Type: BuiltIn, Key: "root"}

type builtinService struct{}

var _ Service = (*builtinService)(nil)

func (b *builtinService) Schema() *Schema {
	return &Schema{
		Type:   BuiltIn,
		Fields: map[string]schema.Field{},
	}
}

func (b *builtinService) RetrieveEntity(key string) (Entity, error) {
	switch key {
	case "root":
		return Entity{Name: "root"}, nil
	default:
		panic("[ontology] - builtin entity not found")
	}
}
