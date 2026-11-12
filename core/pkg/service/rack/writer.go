// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create, update, and delete racks within a Synnax cluster.
type Writer struct {
	// tx is the underlying gorp transaction that rack operations will be executed
	// against.
	tx gorp.Tx
	// newKey returns a new key for a rack.
	newKey func(context.Context) (Key, error)
	// status is used to write status updates.
	status status.Writer
	// statuses retrieves the status that a rack holds.
	statuses *status.Service
	// table is the gorp table for rack entries.
	table *gorp.Table[Key, Rack]
	// driverRole is the role each new rack receives.
	driverRole role.Key
	// dispatcher attaches the actions of each write to tx.
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// storedDetails returns the details of a stored rack status. A stored status can hold
// any details, and details without a running flag read as not running.
func storedDetails(details any) StatusDetails {
	fields, _ := details.(map[string]any)
	running, _ := fields["running"].(bool)
	return StatusDetails{Running: running}
}

// resolveStatus returns the status to persist for r. The owner selects the status, so a
// provided key is ignored.
func resolveStatus(r *Rack) *Status {
	if r.Status == nil {
		return &Status{
			Owner:   new(r.ResourceID()),
			Name:    r.Name,
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Rack status unknown",
		}
	}
	stat := *r.Status
	stat.Key = ""
	stat.Owner = new(r.ResourceID())
	stat.Name = r.Name
	return &stat
}

// writeStatus sets r.Status to the status that r now holds in the DB. A status provided
// on r replaces the stored status. With none, the stored status stays and takes the
// rack's name. The default status is written only if the rack holds no status, because
// a stored status may be a live status from the Driver.
func (w Writer) writeStatus(ctx context.Context, r *Rack) error {
	if r.Status != nil {
		r.Status = resolveStatus(r)
		return w.status.Set(ctx, r.Status)
	}
	var held []status.Status[any]
	if err := w.statuses.NewRetrieve[any]().
		Where(status.MatchOwners[any](r.ResourceID())).
		Entries(&held).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if len(held) == 0 {
		r.Status = resolveStatus(r)
		return w.status.Set(ctx, r.Status)
	}
	h := held[0]
	r.Status = &Status{
		Key:         h.Key,
		Name:        r.Name,
		Variant:     h.Variant,
		Message:     h.Message,
		Description: h.Description,
		Time:        h.Time,
		Labels:      h.Labels,
		Owner:       h.Owner,
		Details:     storedDetails(h.Details),
	}
	if h.Name == r.Name {
		return nil
	}
	return w.status.Rename(ctx, h.Key, r.Name)
}

// Create creates a rack with the Driver role and attaches a create action that holds
// it. A rack with a zero key gets a new key. Create stores the status provided on the
// rack, or the default "unknown" status when none is provided, and sets r.Status to it.
// Create returns an error wrapping validate.ErrValidation when a rack with the key
// exists.
func (w Writer) Create(ctx context.Context, r *Rack) error {
	var err error
	if r.Key.IsZero() {
		r.Key, err = w.newKey(ctx)
		if err != nil {
			return err
		}
	}
	if err = r.Validate(); err != nil {
		return err
	}
	r.Role = w.driverRole
	if err = group.CheckMember(
		ctx,
		w.tx,
		r.Group,
		resource.TypeRack,
		nil,
	); err != nil {
		return err
	}
	if err = w.table.NewCreate().Exclusive().Entry(r).Exec(ctx, w.tx); err != nil {
		return err
	}
	r.Status = resolveStatus(r)
	if err = w.status.Set(ctx, r.Status); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, r.Key, NewCreateAction(CreatePayload{Rack: *r}))
	return nil
}

// SetGroup applies a set of the group to each rack with keys, moving it into g, or out
// of its group when g is nil.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	set := NewSetAction(SetPayload{Path: []string{"group"}, Value: g})
	for _, k := range keys {
		if err := w.Apply(ctx, k, set); err != nil {
			return err
		}
	}
	return nil
}

// CreateMany creates the given racks. It returns an error wrapping
// validate.ErrValidation when a rack with one of their keys exists.
func (w Writer) CreateMany(ctx context.Context, racks *[]Rack) error {
	for i := range *racks {
		if err := w.Create(ctx, &(*racks)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Rename applies a set of the name to the rack with the given key. It returns
// query.ErrNotFound if no rack with the key exists and a validation error if name is
// empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.Apply(
		ctx,
		key,
		NewSetAction(SetPayload{Path: []string{"name"}, Value: name}),
	)
}

// Apply applies the actions to the rack with the given key and attaches them to the
// transaction of w. A change of name also renames the rack's status. Apply returns an
// error wrapping validate.ErrValidation when an action is a create, the result fails to
// validate, or the rack's group cannot hold racks, and an error wrapping
// query.ErrNotFound when no rack has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	var prev Rack
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Rack](key)).
		Entry(&prev).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	next, err := Reduce(prev, acts...)
	if err != nil {
		return err
	}
	if err = next.Validate(); err != nil {
		return err
	}
	if err = group.CheckMember(
		ctx,
		w.tx,
		next.Group,
		resource.TypeRack,
		nil,
	); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(&next).Exec(ctx, w.tx); err != nil {
		return err
	}
	if next.Name != prev.Name {
		if err = w.writeStatus(ctx, &next); err != nil {
			return err
		}
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// Delete deletes the racks with keys, their statuses, and every task they hold. It
// returns a validation error when a rack is embedded, holds a device, or fails guard.
// Delete racks through task.Writer.DeleteRacks, which keeps the tasks of users.
func (w Writer) Delete(
	ctx context.Context,
	guard gorp.GuardFunc[Key, Rack],
	keys ...Key,
) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Rack](keys...)).
		Guard(guardEmbedded).
		Guard(guard).
		Exec(ctx, w.tx)
}

func guardEmbedded(_ gorp.Context, r Rack) error {
	if !r.Embedded {
		return nil
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"cannot delete embedded rack %s",
		r.Name,
	)
}
