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
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

// LabelsOntologyTraverser is an ontology.Traverser that allows the caller to traverse
// an ontology.Retrieve query to find all the labels for a particular resource. Pass
// this traverser to ontology.Retrieve.TraverseTo.
var LabelsOntologyTraverser = ontology.Traverser{
	Traverse: func(_ []ontology.ID) ontology.RawTraversal {
		return func(data []byte, nextIDs *[]ontology.ID) {
			*nextIDs = append(
				*nextIDs,
				ontology.ReadRawID(orc.NewRaw(data).SkipStrings(3)),
			)
		}
	},
	Direction:    ontology.DirectionForward,
	FilterPrefix: ontology.RelationshipPrefix(OntologyRelationshipTypeLabeledBy),
}

// OntologyID constructs a unique ontology.ID for the label with the given key.
func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeLabel, Key: k.String()}
}

// OntologyIDs constructs a slice of unique ontology.IDs for the labels with the given
// keys.
func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(k Key, _ int) ontology.ID { return OntologyID(k) })
}

// OntologyIDsFromLabels constructs a slice of unique ontology.IDs for the given labels.
func OntologyIDsFromLabels(labels []Label) []ontology.ID {
	return lo.Map(labels, func(l Label, _ int) ontology.ID { return OntologyID(l.Key) })
}

// KeysFromOntologyIDs extracts the label keys from the given ontology.IDs.
func KeysFromOntologyIDs(ids []ontology.ID) ([]Key, error) {
	keys := make([]Key, len(ids))
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
	"color": color.Schema,
})

func newResource(l Label) ontology.Resource {
	return ontology.NewResource(schema, OntologyID(l.Key), l.Name, l)
}

type change = xchange.Change[Key, Label]

var (
	_ ontology.Service = (*Service)(nil)
	_ search.Service   = (*Service)(nil)
)

func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeLabel }

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
		Key:     OntologyID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Seq[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Label]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter(ctx context.Context) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := s.table.OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
