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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
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
		"key":       {Type: schema.String},
		"name":      {Type: schema.String},
		"node_key":  {Type: schema.Uint32},
		"rate":      {Type: schema.Float64},
		"is_index":  {Type: schema.Bool},
		"index":     {Type: schema.String},
		"data_type": {Type: schema.String},
	},
}

func newResource(c Channel) schema.Resource {
	e := schema.NewResource(_schema, c.Name)
	schema.Set(e, "key", c.Key().String())
	schema.Set(e, "name", c.Name)
	schema.Set(e, "node_key", uint32(c.NodeKey))
	schema.Set(e, "rate", float64(c.Rate))
	schema.Set(e, "is_index", c.IsIndex)
	schema.Set(e, "index", c.Index().String())
	schema.Set(e, "data_type", string(c.DataType))
	return e
}

var _ ontology.Service = (*service)(nil)

// Schema implements ontology.Service.
func (s *service) Schema() *schema.Schema { return _schema }

// RetrieveResource implements ontology.Service.
func (s *service) RetrieveResource(ctx context.Context, key string) (schema.Resource, error) {
	k, err := ParseKey(key)
	if err != nil {
		return schema.Resource{}, err
	}
	var ch Channel
	return newResource(ch), s.NewRetrieve().WhereKeys(k).Entry(&ch).Exec(ctx, nil)
}

// OnChange implements ontology.Service.
func (s *service) OnChange(f func(context.Context, iter.Next[schema.Resource])) {
	gorp.Observe[Key, Channel](s.DB).OnChange(
		func(ctx context.Context, reader gorp.TxReader[Key, Channel]) {
			f(ctx, newNextCloser(iter.NopNextCloser[Channel]{Wrap: reader.Sets()}))
		})
}

// OpenNext implements ontology.service.
func (s *service) OpenNext() iter.NextCloser[schema.Resource] {
	return newNextCloser(gorp.WrapReader[Key, Channel](s.DB).OpenNext())
}

func newNextCloser(i iter.NextCloser[Channel]) iter.NextCloser[schema.Resource] {
	return iter.NextCloserTranslator[Channel, schema.Resource]{
		Wrap:      i,
		Translate: newResource,
	}
}
