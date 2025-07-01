// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"
)

const OntologyType ontology.Type = "range"

// OntologyID returns the unique ID to identify the range within the Synnax ontology.
func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: OntologyType, Key: k.String()}
}

// OntologyIDs converts a slice of keys to a slice of ontology IDs.
func OntologyIDs(keys []uuid.UUID) (ids []ontology.ID) {
	return lo.Map(keys, func(k uuid.UUID, _ int) ontology.ID {
		return OntologyID(k)
	})
}

func KeyFromOntologyID(id ontology.ID) (uuid.UUID, error) {
	return uuid.Parse(id.Key)
}

// KeysFromOntologyIDs converts a slice of ontology IDs to a slice of keys, returning
// an error if any of the IDs are invalid.
func KeysFromOntologyIDs(ids []ontology.ID) (keys []uuid.UUID, err error) {
	keys = make([]uuid.UUID, len(ids))
	for i, id := range ids {
		keys[i], err = KeyFromOntologyID(id)
		if err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// OntologyIDsFromRanges converts a slice of ranges to a slice of ontology IDs.
func OntologyIDsFromRanges(ranges []Range) (ids []ontology.ID) {
	return lo.Map(ranges, func(r Range, _ int) ontology.ID {
		return OntologyID(r.Key)
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":        zyn.UUID(),
	"name":       zyn.String(),
	"time_range": telem.TimeRangeSchema,
})

func newResource(r Range) ontology.Resource {
	return core.NewResource(schema, OntologyID(r.Key), r.Name, r)
}

var _ ontology.Service = (*Service)(nil)

type change = changex.Change[uuid.UUID, Range]

func (s *Service) Type() ontology.Type { return OntologyType }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k := uuid.MustParse(key)
	var r Range
	err := s.NewRetrieve().WhereKeys(k).Entry(&r).Exec(ctx, tx)
	return newResource(r), err
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(ctx context.Context, nexter iter.Nexter[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Range]) {
		f(ctx, iter.NexterTranslator[change, ontology.Change]{Wrap: reader, Translate: translateChange})
	}
	return gorp.Observe[uuid.UUID, Range](s.DB).OnChange(handleChange)
}

// OpenNexter implements ontology.Service.
func (s *Service) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, Range](s.DB).OpenNexter()
	return iter.NexterCloserTranslator[Range, ontology.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, err
}
