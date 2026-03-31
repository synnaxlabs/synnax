// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"context"
	"io"
	"iter"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/zyn"
)

func OntologyID(k uuid.UUID) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeProject, Key: k.String()}
}

func OntologyIDs(keys []uuid.UUID) []ontology.ID {
	return lo.Map(keys, func(k uuid.UUID, _ int) ontology.ID { return OntologyID(k) })
}

func OntologyIDsFromProjects(projects []Project) []ontology.ID {
	return lo.Map(projects, func(p Project, _ int) ontology.ID {
		return OntologyID(p.Key)
	})
}

func KeysFromOntologyIDs(ids []ontology.ID) ([]uuid.UUID, error) {
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
	"key":  zyn.UUID(),
	"name": zyn.String(),
})

func newResource(p Project) ontology.Resource {
	return ontology.NewResource(schema, OntologyID(p.Key), p.Name, p)
}

type change = xchange.Change[uuid.UUID, Project]

var (
	_ ontology.Service = (*Service)(nil)
	_ search.Service   = (*Service)(nil)
)

func (s *Service) Type() ontology.ResourceType { return ontology.ResourceTypeProject }

// Schema implements ontology.Service.
func (s *Service) Schema() zyn.Schema { return schema }

// RetrieveResource implements ontology.Service.
func (s *Service) RetrieveResource(ctx context.Context, key string, tx gorp.Tx) (ontology.Resource, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return ontology.Resource{}, err
	}
	var p Project
	if err = s.NewRetrieve().WhereKeys(k).Entry(&p).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(p), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     OntologyID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(f func(context.Context, iter.Seq[ontology.Change])) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[uuid.UUID, Project]) {
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
