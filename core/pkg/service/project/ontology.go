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
	"iter"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	xiter "github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

func ResourceID(k Key) resource.ID {
	return resource.ID{Type: resource.TypeProject, Key: k.String()}
}

func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

func ResourceIDsFromProjects(projects []Project) []resource.ID {
	return lo.Map(projects, func(p Project, _ int) resource.ID {
		return p.ResourceID()
	})
}

// KeyFromResourceID converts id to the Key of the project it identifies. It returns an
// error wrapping validate.ErrValidation if id is not a project or its key is not a
// valid UUID. Callers that resolve a caller-supplied field scope the returned error to
// that field with validate.PathedError.
func KeyFromResourceID(id resource.ID) (Key, error) {
	if id.Type != resource.TypeProject {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation, "must be a project, got %q", id.Type,
		)
	}
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return uuid.Nil(), errors.Wrapf(
			validate.ErrValidation, "invalid project key %q: %v", id.Key, err,
		)
	}
	return key, nil
}

var schema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"name": zyn.String(),
})

func newResource(p Project) ontology.Resource {
	return ontology.NewResource(schema, p.ResourceID(), p.Name, p)
}

type change = xchange.Change[Key, Project]

var _ ontology.Service = (*Service)(nil)

func (s *Service) Type() resource.Type { return resource.TypeProject }

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
	var p Project
	if err = s.NewRetrieve().Where(MatchKeys(k)).Entry(&p).Exec(ctx, tx); err != nil {
		return ontology.Resource{}, err
	}
	return newResource(p), nil
}

func translateChange(c change) ontology.Change {
	return ontology.Change{
		Variant: c.Variant,
		Key:     ResourceID(c.Key).String(),
		Value:   newResource(c.Value),
	}
}

// OnChange implements ontology.Service.
func (s *Service) OnChange(
	f func(context.Context, iter.Seq[ontology.Change]),
) observe.Disconnect {
	handleChange := func(ctx context.Context, reader gorp.TxReader[Key, Project]) {
		f(ctx, xiter.Map(reader, translateChange))
	}
	return s.table.Observe().OnChange(handleChange)
}

// Search implements ontology.Service.
func (s *Service) Search(
	ctx context.Context,
	tx gorp.Tx,
	q search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	return resource.SearchIndex(ctx, tx, s.indexes.search, ResourceID, q)
}
