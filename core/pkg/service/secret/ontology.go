// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"context"
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	xchange "github.com/synnaxlabs/x/change"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/zyn"
)

// OntologyID returns the ontology ID of the secret with the given key.
func OntologyID(key Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeSecret, Key: key.String()}
}

// OntologyIDs returns the ontology IDs of the secrets with the given keys.
func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(k Key, _ int) ontology.ID { return OntologyID(k) })
}

// OntologyIDsFromSecrets returns the ontology IDs of the given secrets.
func OntologyIDsFromSecrets(secrets []Secret) []ontology.ID {
	return lo.Map(secrets, func(s Secret, _ int) ontology.ID { return s.OntologyID() })
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":    zyn.UUID(),
	"name":   zyn.String(),
	"source": zyn.String(),
})

func newResource(s Secret) ontology.Resource {
	base := s.Base()
	return ontology.NewResource(schema, s.OntologyID(), base.Name, map[string]any{
		"key":    base.Key,
		"name":   base.Name,
		"source": string(s.Source()),
	})
}

var _ ontology.Service = (*Service)(nil)

type change = xchange.Change[Key, Secret]

// Type implements ontology.Service.
func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeSecret }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var sec Secret
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&sec).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(sec), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Secret]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

func newSearchIndex() *gorp.SearchIndex[Key, Secret] {
	return gorp.NewSearchIndex("search", func(e *Secret) []string {
		return []string{e.Base().Name}
	})
}

// Search implements ontology.Service.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	term string,
	limit int,
) (search.Result[ontology.ID], error) {
	return ontology.SearchIndex(ctx, tx, s.search, term, limit, OntologyID)
}
