// Copyright 2026 Synnax Labs, Inc.
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
	"io"
	"iter"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const OntologyType ontology.Type = "label"

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
	return ontology.ID{Type: OntologyType, Key: k.String()}
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
func KeysFromOntologyIds(ids []ontology.ID) ([]uuid.UUID, error) {
	keys := make([]uuid.UUID, len(ids))
	var err error
	for i, id := range ids {
		if keys[i], err = uuid.Parse(id.Key); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":   zyn.UUID(),
	"name":  zyn.String(),
	"color": zyn.String(),
})

func newResource(l Label) ontology.Resource {
	return ontology.NewResource(schema, OntologyID(l.Key), l.Name, l)
}

type change = xchange.Change[uuid.UUID, Label]

func (s *Service) Type() ontology.Type { return OntologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var l Label
	if err = s.NewRetrieve().WhereKeys(k).Entry(&l).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(l), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Seq[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Label]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return gorp.Observe[uuid.UUID, Label](s.cfg.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(ctx context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := gorp.WrapReader[uuid.UUID, Label](s.cfg.DB).OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
