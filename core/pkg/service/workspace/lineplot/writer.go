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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

type Writer struct {
	tx  gorp.Tx
	otg ontology.Writer
}

func (w Writer) Create(
	ctx context.Context,
	ws uuid.UUID,
	p *LinePlot,
) (err error) {
	var exists bool
	if p.Key == uuid.Nil {
		p.Key = uuid.New()
	} else {
		exists, err = gorp.NewRetrieve[uuid.UUID, LinePlot]().WhereKeys(p.Key).Exists(ctx, w.tx)
		if err != nil {
			return
		}
	}
	if err = gorp.NewCreate[uuid.UUID, LinePlot]().Entry(p).Exec(ctx, w.tx); err != nil {
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
		ontology.ParentOf,
		otgID,
	); err != nil {
		return err
	}
	return err
}

func (w Writer) Rename(
	ctx context.Context,
	key uuid.UUID,
	name string,
) error {
	return gorp.NewUpdate[uuid.UUID, LinePlot]().
		WhereKeys(key).
		Change(func(_ gorp.Context, p LinePlot) LinePlot {
			p.Name = name
			return p
		}).Exec(ctx, w.tx)
}

func (w Writer) SetData(
	ctx context.Context,
	key uuid.UUID,
	data map[string]any,
) error {
	return gorp.NewUpdate[uuid.UUID, LinePlot]().
		WhereKeys(key).
		Change(func(_ gorp.Context, p LinePlot) LinePlot {
			p.Data = data
			return p
		}).Exec(ctx, w.tx)
}

func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	err := gorp.NewDelete[uuid.UUID, LinePlot]().WhereKeys(keys...).Exec(ctx, w.tx)
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
