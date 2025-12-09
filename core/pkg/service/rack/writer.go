// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

// Writer is used to create, update, and delete racks within a Synnax cluster.
type Writer struct {
	// tx is the underlying gorp transaction that rack operations will be executed
	// against.
	tx gorp.Tx
	// otg is a writer used to modify rack related resources and relationships within
	// the ontology.
	otg ontology.Writer
	// group is the base group that racks will be created under.
	group group.Group
	// newKey returns a new key for a rack.
	newKey func() (Key, error)
	// newTaskKey returns a new key for a task within the rack.
	newTaskKey func(context.Context, Key) (uint32, error)
	// status is used to write status updates.
	status status.Writer[StatusDetails]
}

func resolveStatus(r *Rack) *Status {
	if r.Status == nil {
		return &Status{
			Key:     OntologyID(r.Key).String(),
			Name:    r.Name,
			Time:    telem.Now(),
			Variant: xstatus.WarningVariant,
			Message: "Status unknown",
			Details: StatusDetails{Rack: r.Key},
		}
	}
	r.Status.Key = OntologyID(r.Key).String()
	r.Status.Details.Rack = r.Key
	r.Status.Name = r.Name
	return r.Status
}

// Create creates or updates a rack. If the rack key is zero or a rack with the key
// does not exist, a new rack will be created. If a status is provided on the rack,
// it will be used instead of the default "unknown" status.
func (w Writer) Create(ctx context.Context, r *Rack) (err error) {
	if r.Key.IsZero() {
		r.Key, err = w.newKey()
		if err != nil {
			return
		}
	}
	if err = r.Validate(); err != nil {
		return err
	}
	if err = gorp.NewCreate[Key, Rack]().Entry(r).Exec(ctx, w.tx); err != nil {
		return
	}
	otgID := OntologyID(r.Key)
	if err = w.otg.DefineResource(ctx, otgID); err != nil {
		return err
	}
	stat := resolveStatus(r)
	if err = w.status.Set(ctx, stat); err != nil {
		return err
	}
	return w.otg.DefineRelationship(ctx, w.group.OntologyID(), ontology.ParentOf, otgID)
}

// Delete deletes the rack with the provided key and its associated status. Delete is
// idempotent, and deleting a non-existent rack will not return an error.
func (w Writer) Delete(ctx context.Context, key Key) error {
	return w.DeleteGuard(ctx, key, nil)
}

// DeleteGuard deletes the rack with the given key and its associated status if the
// provided guard function returns nil.
func (w Writer) DeleteGuard(ctx context.Context, key Key, guard gorp.GuardFunc[Key, Rack]) error {
	if err := gorp.NewDelete[Key, Rack]().WhereKeys(key).Guard(guard).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.status.Delete(ctx, OntologyID(key).String())
}

// NewTaskKey returns a new, unique key for the task on the provided rack.
func (w Writer) NewTaskKey(ctx context.Context, key Key) (next uint32, err error) {
	return w.newTaskKey(ctx, key)
}
