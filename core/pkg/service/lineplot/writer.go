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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx         gorp.Tx
	otg        ontology.Writer
	table      *gorp.Table[Key, LinePlot]
	dispatcher actions.Dispatcher[Key, Action]
}

func (w Writer) Create(ctx context.Context, projectKey project.Key, lp *LinePlot) error {
	var (
		exists bool
		err    error
	)
	if lp.Key == uuid.Nil {
		lp.Key = uuid.New()
	} else {
		exists, err = w.table.NewRetrieve().Where(gorp.MatchKeys[Key, LinePlot](lp.Key)).Exists(ctx, w.tx)
		if err != nil {
			return err
		}
	}
	// Materialize lines for any channel/range bindings supplied at creation so a plot
	// created with channels and ranges but no lines is fully populated.
	lp.Lines = reconcileLines(*lp)
	lp.ApplyDefaults()
	if err := lp.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(lp).Exec(ctx, w.tx); err != nil {
		return err
	}
	if exists {
		return nil
	}
	otgID := lp.OntologyID()
	if err := w.otg.DefineResources(ctx, otgID); err != nil {
		return err
	}
	if projectKey == uuid.Nil {
		return nil
	}
	return w.otg.DefineRelationships(
		ctx,
		project.OntologyID(projectKey),
		ontology.RelationshipTypeParentOf,
		otgID,
	)
}

// CreateMany creates the given line plots within the project provided. If line plots
// with the same key already exist, they will be overwritten.
func (w Writer) CreateMany(
	ctx context.Context,
	projectKey project.Key,
	plots *[]LinePlot,
) error {
	for i := range *plots {
		if err := w.Create(ctx, projectKey, &(*plots)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Dispatch applies a sequence of actions atomically to the line plot with the given
// key. After a successful update the actions are notified to the service-level observer
// so subscribers (cluster signals) can broadcast them. dispatchKey is a
// client-generated identifier carried verbatim onto the broadcast so the originating
// client can match its own echo against the set of outstanding local replays and skip a
// redundant reduce when no foreign action interleaved.
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

func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.table.NewDelete().Where(gorp.MatchKeys[Key, LinePlot](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResources(ctx, OntologyIDs(keys)...)
}
