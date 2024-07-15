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
	"github.com/synnaxlabs/x/observe"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	changex "github.com/synnaxlabs/x/change"
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
		"key":       {Type: schema.Uint32},
		"name":      {Type: schema.String},
		"node_key":  {Type: schema.Uint32},
		"rate":      {Type: schema.Float64},
		"is_index":  {Type: schema.Bool},
		"index":     {Type: schema.String},
		"data_type": {Type: schema.String},
		"internal":  {Type: schema.Bool},
	},
}

func newResource(c Channel) schema.Resource {
	e := schema.NewResource(_schema, OntologyID(c.Key()), c.Name)
	schema.Set(e, "key", uint32(c.Key()))
	schema.Set(e, "name", c.Name)
	schema.Set(e, "node_key", uint32(c.Leaseholder))
	schema.Set(e, "rate", float64(c.Rate))
	schema.Set(e, "is_index", c.IsIndex)
	schema.Set(e, "index", c.Index().String())
	schema.Set(e, "data_type", string(c.DataType))
	schema.Set(e, "internal", c.Internal)
	return e
}

var _ ontology.Service = (*service)(nil)

type change = changex.Change[Key, Channel]

// Schema implements ontology.Service.
func (s *service) Schema() *schema.Schema { return _schema }

// RetrieveResource implements ontology.Service.
func (s *service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k := MustParseKey(key)
	var ch Channel
	err := s.NewRetrieve().WhereKeys(k).Entry(&ch).Exec(ctx, tx)
	return newResource(ch), err
}

func translateChange(ch change) schema.Change {
	return schema.Change{
		Variant: ch.Variant,
		Key:     OntologyID(ch.Key),
		Value:   newResource(ch.Value),
	}
}

// OnChange implements ontology.Service.
func (s *service) OnChange(f func(context.Context, iter.Nexter[schema.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Channel]) {
		f(ctx, iter.NexterTranslator[change, schema.Change]{
			Wrap:      reader,
			Translate: translateChange,
		})
	}
	return gorp.Observe[Key, Channel](s.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *service) OpenNexter() (iter.NexterCloser[schema.Resource], error) {
	n, err := gorp.WrapReader[Key, Channel](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[Channel, schema.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
