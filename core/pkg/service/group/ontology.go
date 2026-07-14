// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"context"
	"io"
	"iter"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
)

func OntologyID(key Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeGroup, Key: key.String()}
}

func OntologyIDs(keys []Key) []ontology.ID {
	return lo.Map(keys, func(k Key, _ int) ontology.ID { return OntologyID(k) })
}

func OntologyIDsFromGroups(groups []Group) []ontology.ID {
	return lo.Map(groups, func(g Group, _ int) ontology.ID { return OntologyID(g.Key) })
}

func newResource(g Group) ontology.Resource {
	return ontology.NewResource(schema, OntologyID(g.Key), g.Name, g)
}

type change = xchange.Change[Key, Group]

var (
	_ ontology.Service = (*Service)(nil)
	_ search.Service   = (*Service)(nil)
)

func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeGroup }

func (s *Service) RetrieveResource(
	ctx context.Context,
	key string,
	tx gorp.Tx,
) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var g Group
	if err = s.NewRetrieve().Entry(&g).Where(MatchKeys(k)).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(g), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Group]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

func (s *Service) OpenNexter(
	ctx context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	n, closer, err := s.table.OpenNexter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return xiter.Map(n, newResource), closer, nil
}
