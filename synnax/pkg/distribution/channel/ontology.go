package channel

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
)

const ontologyType ontology.Type = "channel"

// OntologyID returns a unique identifier for a Channel for use within a resource
// ontology.
func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

var _schema = &ontology.Schema{
	Type: ontologyType,
	Fields: map[string]schema.Field{
		"key":     {Type: schema.String},
		"name":    {Type: schema.String},
		"nodeID":  {Type: schema.Uint32},
		"rate":    {Type: schema.Float64},
		"Density": {Type: schema.Uint32},
	},
}

var _ ontology.Service = (*Service)(nil)

func (s *Service) Schema() *schema.Schema { return _schema }

func (s *Service) RetrieveEntity(key string) (schema.Entity, error) {
	k, err := ParseKey(key)
	if err != nil {
		return schema.Entity{}, err
	}
	var ch Channel
	err = s.NewRetrieve().WhereKeys(k).Entry(&ch).Exec(context.TODO())
	return newEntity(ch), err
}

func newEntity(c Channel) schema.Entity {
	e := schema.NewEntity(_schema, c.Name)
	schema.Set(e, "key", c.Key().String())
	schema.Set(e, "name", c.Name)
	schema.Set(e, "nodeID", uint32(c.NodeID))
	schema.Set(e, "rate", float64(c.Rate))
	schema.Set(e, "Density", uint32(c.Density))
	return e
}
