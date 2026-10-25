// Copyright 2026 Synnax Labs, Inc.
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
	"iter"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xchange "github.com/synnaxlabs/x/change"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyID returns a unique identifier for a Channel for use within a resource
// ontology.
func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeChannel, Key: k.String()}
}

// OntologyIDsFromKeys returns the ontology.ID for each key.
func OntologyIDsFromKeys(keys Keys) []ontology.ID {
	return lo.Map(keys, func(k Key, _ int) ontology.ID { return OntologyID(k) })
}

// KeysFromOntologyIDs returns a slice of Keys from a slice of ontology.IDs.
func KeysFromOntologyIDs(ids []ontology.ID) (Keys, error) {
	return lo.MapErr(ids, func(id ontology.ID, _ int) (Key, error) {
		return ParseKey(id.Key)
	})
}

// OntologyIDsFromChannels returns the ontology.ID for each channel.
func OntologyIDsFromChannels(chs []Channel) []ontology.ID {
	return lo.Map(chs, func(item Channel, _ int) ontology.ID {
		return OntologyID(item.Key())
	})
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":         zyn.Uint32().Coerce(),
	"name":        zyn.String(),
	"leaseholder": zyn.Uint16().Coerce(),
	"is_index":    zyn.Bool(),
	"index":       zyn.Uint32().Coerce(),
	"data_type":   zyn.String(),
	"internal":    zyn.Bool(),
	"virtual":     zyn.Bool(),
	"expression":  zyn.String(),
})

func newResource(c Channel) ontology.Resource {
	return ontology.NewResource(schema, OntologyID(c.Key()), c.Name, c.ToPayload())
}

var _ ontology.Service = (*Service)(nil)

type change = xchange.Change[Key, Channel]

// Type implements ontology.Service.
func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeChannel }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context, key string, tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := ParseKey(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var ch Channel
	if err := s.NewRetrieve().Where(MatchKeys(k)).Entry(&ch).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(ch), nil
}

func translateChange(ch change) ontology.Change {
	return ontology.Change{
		Variant: ch.Variant,
		Key:     OntologyID(ch.Key).String(),
		Value:   newResource(ch.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Channel]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.Observe().OnChange(handleChange)
}

// Search implements ontology.Service.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	term string,
	limit int,
) (search.Result[ontology.ID], error) {
	return ontology.SearchIndex(ctx, tx, s.indexes.search, term, limit, OntologyID)
}
