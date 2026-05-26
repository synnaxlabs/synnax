// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx         gorp.Tx
	otg        ontology.Writer
	table      *gorp.Table[Key, LinePlot]
	dispatcher actions.Dispatcher[Key, Action]
}

func (w Writer) Create(
	ctx context.Context,
	ws workspace.Key,
	p *LinePlot,
) (err error) {
	var exists bool
	if p.Key == uuid.Nil {
		p.Key = uuid.New()
	} else {
		exists, err = w.table.NewRetrieve().Where(gorp.MatchKeys[Key, LinePlot](p.Key)).Exists(ctx, w.tx)
		if err != nil {
			return
		}
	}
	if err = w.table.NewCreate().Entry(p).Exec(ctx, w.tx); err != nil {
		return
	}
	if exists {
		return
	}
	otgID := OntologyID(p.Key)
	if err := w.otg.DefineResource(ctx, otgID); err != nil {
		return err
	}
	if err := w.otg.DefineRelationship(
		ctx,
		workspace.OntologyID(ws),
		ontology.RelationshipTypeParentOf,
		otgID,
	); err != nil {
		return err
	}
	return err
}

func (w Writer) Rename(
	ctx context.Context,
	key Key,
	name string,
) error {
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, LinePlot](key)).
		Change(func(_ gorp.Context, p LinePlot) LinePlot {
			p.Name = name
			return p
		}).Exec(ctx, w.tx)
}

// SetData replaces the body of the line plot with the given key with the
// provided value. Key and Name are preserved from the existing entry; every
// other field on data overwrites the stored entry verbatim.
func (w Writer) SetData(
	ctx context.Context,
	key Key,
	data LinePlot,
) error {
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, LinePlot](key)).
		Change(func(_ gorp.Context, p LinePlot) LinePlot {
			data.Key = p.Key
			data.Name = p.Name
			return data
		}).Exec(ctx, w.tx)
}

// Dispatch applies a sequence of actions atomically to the line plot with the
// given key. After a successful update the actions are notified to the
// service-level observer so subscribers (cluster signals) can broadcast them.
// dispatchKey is a client-generated identifier carried verbatim onto the
// broadcast so the originating client can match its own echo against the set
// of outstanding local replays and skip a redundant reduce when no foreign
// action interleaved.
func (w Writer) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	actions []Action,
) error {
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, LinePlot](key)).
		ChangeErr(func(_ gorp.Context, p LinePlot) (LinePlot, error) {
			return Reduce(p, actions...)
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Notify(ctx, key, dispatchKey, actions)
	return nil
}

func (w Writer) Delete(
	ctx context.Context,
	keys ...Key,
) error {
	err := w.table.NewDelete().Where(gorp.MatchKeys[Key, LinePlot](keys...)).Exec(ctx, w.tx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return nil
}
