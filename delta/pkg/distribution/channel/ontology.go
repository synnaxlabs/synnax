package channel

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/ontology"
	"github.com/arya-analytics/delta/pkg/distribution/ontology/schema"
)

const ontologyType ontology.Type = "channel"

var _schema = &ontology.Schema{
	Type: ontologyType,
	Fields: map[string]schema.Field{
		"key":     {Type: schema.String},
		"name":    {Type: schema.String},
		"nodeID":  {Type: schema.Uint32},
		"rate":    {Type: schema.Float64},
		"Density": {Type: schema.Uint16},
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
	return newEntity(ch), s.NewRetrieve().WhereKeys(k).Entry(&ch).Exec(context.TODO())
}

func newEntity(c Channel) schema.Entity {
	e := schema.NewEntity(_schema)
	schema.Set(e, "key", c.Key().String())
	schema.Set(e, "name", c.Name)
	schema.Set(e, "nodeID", int32(c.NodeID))
	schema.Set(e, "rate", float64(c.Rate))
	schema.Set(e, "Density", float32(c.Density))
	return e
}
