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

package task

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	rack  rack.Writer
	group group.Group
}

func (w Writer) Create(ctx context.Context, r *Task) (err error) {
	if !r.Key.IsValid() {
		localKey, err := w.rack.IncrementModuleCount(ctx, r.Rack(), 1)
		if err != nil {
			return err
		}
		r.Key = NewKey(r.Rack(), localKey)
	}
	r.State = nil
	// We don't create ontology resources for internal tasks.
	if err = gorp.NewCreate[Key, Task]().Entry(r).Exec(ctx, w.tx); err != nil || r.Internal {
		return
	}
	otgID := OntologyID(r.Key)
	if err = w.otg.DefineResource(ctx, otgID); err != nil {
		return err
	}
	return w.otg.DefineRelationship(ctx, w.group.OntologyID(), ontology.ParentOf, otgID)
}

func (w Writer) Delete(ctx context.Context, key Key, allowInternal bool) error {
	q := gorp.NewDelete[Key, Task]().WhereKeys(key)
	if !allowInternal {
		q = q.Guard(func(t Task) error {
			if t.Internal {
				return errors.Wrapf(validate.Error, "internal task %v cannot be deleted", t)
			}
			return nil
		})
	}
	if err := q.Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResource(ctx, OntologyID(key))
}
