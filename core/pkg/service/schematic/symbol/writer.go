// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Writer is used to create, update, and delete symbols within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	svc       *Service
	tx        gorp.Tx
	otgWriter ontology.Writer
	table     *gorp.Table[Key, Symbol]
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

// Create creates the given symbol. If the symbol does not have a key, a new key will be
// generated. An existing symbol keeps its group, and a new symbol takes s.Group.
func (w Writer) Create(ctx context.Context, s *Symbol) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, s) })
}

func (w Writer) create(ctx context.Context, s *Symbol) error {
	s.ApplyDefaults()
	if err := s.Validate(); err != nil {
		return err
	}
	var exists bool
	if s.Key == uuid.Nil() {
		s.Key = uuid.New()
	} else {
		var stored Symbol
		err := w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Symbol](s.Key)).
			Entry(&stored).
			Exec(ctx, w.tx)
		if errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		if exists = err == nil; exists {
			s.Group = stored.Group
		}
	}
	if !exists {
		if err := group.CheckMember(
			ctx,
			w.tx,
			s.Group,
			resource.TypeSchematicSymbol,
			nil,
		); err != nil {
			return err
		}
	}
	if err := w.table.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DefineResources(ctx, s.ResourceID())
}

// CreateMany creates the given symbols. If symbols with the same key already exist,
// they will be overwritten.
func (w Writer) CreateMany(ctx context.Context, symbols *[]Symbol) error {
	for i := range *symbols {
		if err := w.Create(ctx, &(*symbols)[i]); err != nil {
			return err
		}
	}
	return nil
}

// SetGroup moves the symbols with keys into g, or out of their group when g is nil.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	if err := group.CheckMember(
		ctx,
		w.tx,
		g,
		resource.TypeSchematicSymbol,
		nil,
	); err != nil {
		return err
	}
	return setGroup(w.table, g, keys...).Exec(ctx, w.tx)
}

// Rename renames the symbol with the given key to the provided name.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Symbol](key)).
		Change(func(_ gorp.Context, s Symbol) Symbol {
			s.Name = name
			return s
		}).
		Exec(ctx, w.tx)
}

// Delete deletes the symbols with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Symbol](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, ResourceIDs(keys)...)
}
