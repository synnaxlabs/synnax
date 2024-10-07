/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package rack

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx     gorp.Tx
	otg    ontology.Writer
	newKey func() (Key, error)
}

func (w Writer) Create(ctx context.Context, r *Rack) (err error) {
	if !r.Key.IsValid() {
		r.Key, err = w.newKey()
		if err != nil {
			return
		}
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if err = gorp.NewCreate[Key, Rack]().Entry(r).Exec(ctx, w.tx); err != nil {
		return
	}
	otgID := OntologyID(r.Key)
	return w.otg.DefineResource(ctx, otgID)
}

func (w Writer) Delete(ctx context.Context, key Key) error {
	if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
		return err
	}
	return gorp.NewDelete[Key, Rack]().WhereKeys(key).Exec(ctx, w.tx)
}

func (w Writer) IncrementTaskCount(ctx context.Context, key Key, by uint32) (next uint32, err error) {
	return next, gorp.NewUpdate[Key, Rack]().WhereKeys(key).Change(func(r Rack) Rack {
		r.TaskCounter += by
		next = r.TaskCounter
		return r
	}).Exec(ctx, w.tx)
}
