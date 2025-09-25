// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/core"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

const OntologyType ontology.Type = "group"

func OntologyID(key uuid.UUID) core.ID {
	return ontology.ID{Type: OntologyType, Key: key.String()}
}

func OntologyIDs(keys []uuid.UUID) []core.ID {
	return lo.Map(keys, func(k uuid.UUID, _ int) core.ID { return OntologyID(k) })
}

func newResource(g Group) ontology.Resource {
	return core.NewResource(schema, OntologyID(g.Key), g.Name, g)
}

type change = xchange.Change[uuid.UUID, Group]

func (s *Service) Type() ontology.Type { return OntologyType }

func (s *Service) Schema() zyn.Schema { return schema }

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
	if err = s.NewRetrieve().Entry(&g).WhereKeys(k).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(g), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key),
		Value:   newResource(c.Value),
	}
}

func (s *Service) OnChange(
	f func(context.Context, iter.Nexter[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Group]) {
		f(ctx, iter.NexterTranslator[change, ontology.Change]{
			Wrap:      reader,
			Translate: translateChange,
		})
	}
	return gorp.Observe[uuid.UUID, Group](s.DB).OnChange(handleChange)
}

func (s *Service) OpenNexter() (iter.NexterCloser[ontology.Resource], error) {
	n, err := gorp.WrapReader[uuid.UUID, Group](s.DB).OpenNexter()
	if err != nil {
		return nil, err
	}
	return iter.NexterCloserTranslator[Group, ontology.Resource]{
		Wrap:      n,
		Translate: newResource,
	}, nil
}
