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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	svc   *Service
	tx    gorp.Tx
	otg   ontology.Writer
	table *gorp.Table[Key, Project]
}

// inTx calls f with w, or with a Writer on a transaction of its own when w has no
// transaction.
func (w Writer) inTx(ctx context.Context, f func(Writer) error) error {
	db, ok := w.tx.(*gorp.DB)
	if !ok {
		return f(w)
	}
	return db.WithTx(ctx, func(tx gorp.Tx) error { return f(w.svc.NewWriter(tx)) })
}

// Create creates p, or updates the stored project with p.Key. An existing project keeps
// its group.
func (w Writer) Create(ctx context.Context, p *Project) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, p) })
}

func (w Writer) create(
	ctx context.Context,
	p *Project,
) (err error) {
	var exists bool
	if p.Key == uuid.Nil() {
		p.Key = uuid.New()
	} else {
		var stored Project
		err = w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Project](p.Key)).
			Entry(&stored).
			Exec(ctx, w.tx)
		if errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		if exists = err == nil; exists {
			p.Group = stored.Group
		}
	}
	if !exists {
		if err := group.CheckMember(
			ctx,
			w.tx,
			p.Group,
			resource.TypeProject,
			nil,
		); err != nil {
			return err
		}
	}
	if err = p.Validate(); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(p).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DefineResources(ctx, p.ResourceID())
}

// SetGroup moves the projects with keys into g, or out of their group when g is nil.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	return w.inTx(ctx, func(w Writer) error {
		if err := group.CheckMember(
			ctx,
			w.tx,
			g,
			resource.TypeProject,
			nil,
		); err != nil {
			return err
		}
		return setGroup(w.table, g, keys...).Exec(ctx, w.tx)
	})
}

// CreateMany creates the given projects. If projects with the same key already
// exist, they will be overwritten.
func (w Writer) CreateMany(
	ctx context.Context,
	projects *[]Project,
) error {
	for i := range *projects {
		if err := w.Create(ctx, &(*projects)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Rename renames the project with the given key. It returns query.ErrNotFound if no
// project with the key exists and a validation error if name is empty.
func (w Writer) Rename(
	ctx context.Context,
	key Key,
	name string,
) error {
	v := validate.New("project.project")
	v.NotEmptyString("name", name)
	if err := v.Error(); err != nil {
		return err
	}
	return w.table.NewUpdate().Where(gorp.MatchKeys[Key, Project](key)).
		Change(func(_ gorp.Context, p Project) Project {
			p.Name = name
			return p
		}).Exec(ctx, w.tx)
}

func (w Writer) Delete(
	ctx context.Context,
	keys ...Key,
) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Project](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResources(ctx, ResourceIDs(keys)...)
}
