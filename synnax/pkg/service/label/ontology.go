// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
)

const ontologyType ontology.Type = "label"

// Labels is an ontology.Traverser that allows the caller to traverse an ontology.Retrieve
// query to find all the labels for a particular resource. Pass this traverser to
// ontology.Retrieve.TraverseTo.
var (
	Labels = ontology.Traverser{
		Filter: func(res *ontology.Resource, rel *ontology.Relationship) bool {
			return rel.Type == LabeledBy && rel.From == res.ID
		},
		Direction: ontology.Forward,
	}
)

// OntologyID constructs a unique ontology.ID for the label with the given key.
func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: ontologyType, Key: k.String()}
}

// OntologyIDs constructs a slice of unique ontology.IDs for the labels with the given
// keys.
func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(k uuid.UUID, _ int) ontology.ID { return OntologyID(k) })
}

// OntologyIDsFromLabels constructs a slice of unique ontology.IDs for the given labels.
func OntologyIDsFromLabels(labels []Label) []ontology.ID {
	return lo.Map(labels, func(l Label, _ int) ontology.ID { return OntologyID(l.Key) })
}

// KeysFromOntologyIds extracts the label keys from the given ontology.IDs.
func KeysFromOntologyIds(ids []ontology.ID) (keys []uuid.UUID, err error) {
	keys = make([]uuid.UUID, len(ids))
	for i, id := range ids {
		keys[i], err = uuid.Parse(id.Key)
		if err != nil {
			return nil, err
		}
	}
	return keys, nil
}

var _schema = &ontology.Schema{
	Type: ontologyType,
	Fields: map[string]schema.Field{
		"key":   {Type: schema.String},
		"name":  {Type: schema.String},
		"color": {Type: schema.String},
	},
}

func newResource(l Label) schema.Resource {
	e := schema.NewResource(_schema, OntologyID(l.Key), l.Name)
	schema.Set(e, "key", l.Key.String())
	schema.Set(e, "name", l.Name)
	schema.Set(e, "color", string(l.Color))
	return e
}

type change = changex.Change[uuid.UUID, Label]

// Schema implements ontology.Service.
func (s *Service) Schema() *schema.Schema { return _schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k := uuid.MustParse(key)
	var l Label
	err := s.NewRetrieve().WhereKeys(k).Entry(&l).Exec(ctx, tx)
	return newResource(l), err
}

func translateChange(c change) schema.Change {
	return schema.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Nexter[schema.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Label]) {
		f(ctx, iter.NexterTranslator[change, schema.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[uuid.UUID, Label](s.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[schema.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, Label](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[Label, schema.Resource]{Wrap: n, Translate: newResource}, err
}
