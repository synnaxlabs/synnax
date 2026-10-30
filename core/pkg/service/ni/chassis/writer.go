// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package chassis

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Writer creates, updates, and deletes NI chassis. It runs its operations in the
// transaction given to Service.NewWriter, or each in its own when given none.
type Writer struct {
	service *Service
	tx      gorp.Tx
	ownsTx  bool
	otg     ontology.Writer
	status  status.Writer
	table   *gorp.Table[Key, Chassis]
	devices *device.Service
}

func (w Writer) atomic(ctx context.Context, f func(Writer) error) error {
	if !w.ownsTx {
		return f(w)
	}
	return w.service.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return f(w.service.NewWriter(tx))
	})
}

func (w Writer) setParent(ctx context.Context, c Chassis) error {
	id := c.OntologyID()
	if err := w.otg.DeleteIncomingRelationshipsOfType(
		ctx,
		id,
		ontology.RelationshipTypeParentOf,
	); err != nil {
		return err
	}
	return w.otg.DefineRelationships(
		ctx,
		c.Rack.OntologyID(),
		ontology.RelationshipTypeParentOf,
		id,
	)
}

func (w Writer) defineStatus(ctx context.Context, c Chassis) error {
	key := c.OntologyID().String()
	exists, err := gorp.NewRetrieve[status.Key, Status]().
		Where(gorp.MatchKeys[status.Key, Status](key)).
		Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if exists {
		return w.syncStatus(ctx, c)
	}
	return w.status.Set(ctx, &Status{
		Key:     key,
		Name:    c.Name,
		Time:    telem.Now(),
		Variant: status.VariantWarning,
		Message: "Status unknown",
		Details: StatusDetails{Rack: c.Rack, Chassis: c.Key},
	})
}

func (w Writer) syncStatus(ctx context.Context, c Chassis) error {
	var s Status
	if err := gorp.NewRetrieve[status.Key, Status]().
		Where(gorp.MatchKeys[status.Key, Status](c.OntologyID().String())).
		Entry(&s).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	s.Name = c.Name
	s.Details.Rack = c.Rack
	return w.status.Set(ctx, &s)
}

// Create stores c, defines its ontology resource and parent edge, and writes an unknown
// status when it has none. An existing status takes the name and rack of c. A chassis
// with a nil key gets a new one. It returns a
// validation error when c is invalid.
func (w Writer) Create(ctx context.Context, c *Chassis) error {
	if c.Key == uuid.Nil() {
		c.Key = uuid.New()
	}
	if err := c.Validate(); err != nil {
		return err
	}
	return w.atomic(ctx, func(w Writer) error {
		if err := w.table.NewCreate().Entry(c).Exec(ctx, w.tx); err != nil {
			return err
		}
		if err := w.otg.DefineResources(ctx, c.OntologyID()); err != nil {
			return err
		}
		if err := w.setParent(ctx, *c); err != nil {
			return err
		}
		return w.defineStatus(ctx, *c)
	})
}

// Update applies change to the chassis stored under key and stores the result. The
// status and parent edge follow the chassis. It returns an error wrapping
// query.ErrNotFound when no chassis is stored under key and a validation error when the
// result is invalid. A transaction that commits between the read and the write of this
// one is overwritten.
func (w Writer) Update(
	ctx context.Context,
	key Key,
	change func(Chassis) (Chassis, error),
) error {
	return w.atomic(ctx, func(w Writer) error {
		var prev, next Chassis
		if err := w.table.NewUpdate().
			Where(gorp.MatchKeys[Key, Chassis](key)).
			ChangeErr(func(_ gorp.Context, c Chassis) (Chassis, error) {
				prev = c
				updated, err := change(c)
				if err != nil {
					return Chassis{}, err
				}
				updated.Key = key
				if err = updated.Validate(); err != nil {
					return Chassis{}, err
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
		if next.Rack != prev.Rack {
			return w.setParent(ctx, next)
		}
		return nil
	})
}

// Delete removes the chassis stored under keys with their modules, statuses, and
// ontology resources. It returns an error wrapping validate.ErrValidation when a task
// references one of the modules. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.atomic(ctx, func(w Writer) error {
		var modules []device.Device
		if err := w.devices.NewRetrieve().
			Where(device.MatchChassis(keys...)).
			Entries(&modules).
			Exec(ctx, w.tx); err != nil {
			return err
		}
		if err := w.devices.NewWriter(w.tx).Delete(
			ctx,
			lo.Map(modules, func(d device.Device, _ int) device.Key {
				return d.Key
			})...,
		); err != nil {
			return err
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
				Where(gorp.MatchKeys[Key, Chassis](keys...)).
				Exec(ctx, w.tx),
			query.ErrNotFound,
		)
	})
}
