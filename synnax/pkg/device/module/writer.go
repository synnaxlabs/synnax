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

package module

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/device/rack"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx   gorp.Tx
	otg  ontology.Writer
	rack rack.Writer
}

func (w Writer) Create(ctx context.Context, r *Module) (err error) {
	if !r.Key.IsValid() {
		localKey, err := w.rack.IncrementModuleCount(ctx, r.Rack(), 1)
		if err != nil {
			return err
		}
		r.Key = NewKey(r.Rack(), localKey)
	}

	if err = gorp.NewCreate[Key, Module]().Entry(r).Exec(ctx, w.tx); err != nil {
		return
	}
	otgID := OntologyID(r.Key)
	if err := w.otg.DefineResource(ctx, otgID); err != nil {
		return err
	}
	return w.otg.DefineRelationship(ctx, rack.OntologyID(r.Rack()), ontology.ParentOf, otgID)
}

func (w Writer) Delete(ctx context.Context, key Key) error {
	if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
		return err
	}
	return gorp.NewDelete[Key, Module]().WhereKeys(key).Exec(ctx, w.tx)
}
