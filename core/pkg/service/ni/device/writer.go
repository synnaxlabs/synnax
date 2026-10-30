// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, updates, and deletes NI devices. It runs its operations in the
// transaction given to Service.NewWriter, or each in its own when given none.
type Writer struct {
	service    *Service
	tx         gorp.Tx
	ownsTx     bool
	otg        ontology.Writer
	status     status.Writer
	table      *gorp.Table[Key, Device]
	referenced func(ctx context.Context, tx gorp.Tx, key Key) (bool, error)
}

func (w Writer) atomic(ctx context.Context, f func(Writer) error) error {
	if !w.ownsTx {
		return f(w)
	}
	return w.service.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return f(w.service.NewWriter(tx))
	})
}

func parentOf(d Device) ontology.ID {
	if d.Chassis != nil {
		return ontology.ID{
			Type: ontology.ResourceTypeNiChassis,
			Key:  d.Chassis.String(),
		}
	}
	return d.Rack.OntologyID()
}

func (w Writer) setParent(ctx context.Context, d Device) error {
	id := d.OntologyID()
	if err := w.otg.DeleteIncomingRelationshipsOfType(
		ctx,
		id,
		ontology.RelationshipTypeParentOf,
	); err != nil {
		return err
	}
	return w.otg.DefineRelationships(
		ctx,
		parentOf(d),
		ontology.RelationshipTypeParentOf,
		id,
	)
}

func (w Writer) defineStatus(ctx context.Context, d Device) error {
	key := d.OntologyID().String()
	exists, err := gorp.NewRetrieve[status.Key, Status]().
		Where(gorp.MatchKeys[status.Key, Status](key)).
		Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if exists {
		return w.syncStatus(ctx, d)
	}
	return w.status.Set(ctx, &Status{
		Key:     key,
		Name:    d.Name,
		Time:    telem.Now(),
		Variant: status.VariantWarning,
		Message: "Status unknown",
		Details: StatusDetails{Rack: d.Rack, Device: d.Key},
	})
}

func (w Writer) syncStatus(ctx context.Context, d Device) error {
	var s Status
	if err := gorp.NewRetrieve[status.Key, Status]().
		Where(gorp.MatchKeys[status.Key, Status](d.OntologyID().String())).
		Entry(&s).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	s.Name = d.Name
	s.Details.Rack = d.Rack
	return w.status.Set(ctx, &s)
}

// Create stores d, defines its ontology resource and parent edge, and writes an unknown
// status when it has none. An existing status takes the name and rack of d. A device
// with a nil key gets a new one. It returns a
// validation error when d is invalid.
func (w Writer) Create(ctx context.Context, d *Device) error {
	if d.Key == uuid.Nil() {
		d.Key = uuid.New()
	}
	if err := d.Validate(); err != nil {
		return err
	}
	return w.atomic(ctx, func(w Writer) error {
		if err := w.table.NewCreate().Entry(d).Exec(ctx, w.tx); err != nil {
			return err
		}
		if err := w.otg.DefineResources(ctx, d.OntologyID()); err != nil {
			return err
		}
		if err := w.setParent(ctx, *d); err != nil {
			return err
		}
		return w.defineStatus(ctx, *d)
	})
}

// Update applies change to the device stored under key and stores the result, so a
// caller changes one field group and keeps the other as stored. The status and parent
// edge follow the device. It returns an error wrapping query.ErrNotFound when no device
// is stored under key and a validation error when the result is invalid. A transaction
// that commits between the read and the write of this one is overwritten.
func (w Writer) Update(
	ctx context.Context,
	key Key,
	change func(Device) (Device, error),
) error {
	return w.atomic(ctx, func(w Writer) error {
		var prev, next Device
		if err := w.table.NewUpdate().
			Where(gorp.MatchKeys[Key, Device](key)).
			ChangeErr(func(_ gorp.Context, d Device) (Device, error) {
				prev = d
				updated, err := change(d)
				if err != nil {
					return Device{}, err
				}
				updated.Key = key
				if err = updated.Validate(); err != nil {
					return Device{}, err
				}
				next = updated
				return updated, nil
			}).
			Exec(ctx, w.tx); err != nil {
			return err
		}
		if next.Name != prev.Name || next.Rack != prev.Rack {
			if err := w.syncStatus(ctx, next); err != nil {
				return err
			}
		}
		if parentOf(next) != parentOf(prev) {
			return w.setParent(ctx, next)
		}
		return nil
	})
}

// Delete removes the devices stored under keys with their statuses and ontology
// resources. It returns an error wrapping validate.ErrValidation when a task references
// one of them. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.atomic(ctx, func(w Writer) error {
		if w.referenced != nil {
			for _, key := range keys {
				referenced, err := w.referenced(ctx, w.tx, key)
				if err != nil {
					return err
				}
				if referenced {
					return errors.Wrapf(
						validate.ErrValidation,
						"device %s is referenced by a task",
						key,
					)
				}
			}
		}
		ids := OntologyIDs(keys)
		if err := w.status.Delete(ctx, ontology.IDsToKeys(ids)...); err != nil {
			return err
		}
		if err := w.otg.DeleteResources(ctx, ids...); err != nil {
			return err
		}
		return errors.Skip(
			w.table.NewDelete().
				Where(gorp.MatchKeys[Key, Device](keys...)).
				Exec(ctx, w.tx),
			query.ErrNotFound,
		)
	})
}
