package ontology

import "github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"

// BuiltIn is a resource type that is built into the ontology.
const BuiltIn Type = "builtin"

// Root is the root resource in the ontology. All other resources are reachable by
// traversing the ontology from the root.
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
