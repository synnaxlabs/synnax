// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
		"key":      {Type: schema.String},
		"name":     {Type: schema.String},
		"nodeID":   {Type: schema.Uint32},
		"rate":     {Type: schema.Float64},
		"isIndex":  {Type: schema.Bool},
		"index":    {Type: schema.String},
		"dataType": {Type: schema.String},
	},
}

var _ ontology.Service = (*service)(nil)

func (s *service) Schema() *schema.Schema { return _schema }

func (s *service) RetrieveEntity(ctx context.Context, key string) (schema.Entity, error) {
	k, err := ParseKey(key)
	if err != nil {
		return schema.Entity{}, err
	}
	var ch Channel
	err = s.NewRetrieve().WhereKeys(k).Entry(&ch).Exec(ctx)
	return newEntity(ch), err
}

func newEntity(c Channel) schema.Entity {
	e := schema.NewEntity(_schema, c.Name)
	schema.Set(e, "key", c.Key().String())
	schema.Set(e, "name", c.Name)
	schema.Set(e, "nodeID", uint32(c.NodeID))
	schema.Set(e, "rate", float64(c.Rate))
	schema.Set(e, "isIndex", c.IsIndex)
	schema.Set(e, "index", c.Index().String())
	schema.Set(e, "dataType", string(c.DataType))
	return e
}
