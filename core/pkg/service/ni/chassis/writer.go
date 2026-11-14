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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Writer creates, updates, and deletes NI chassis in the transaction given to
// Service.NewWriter. A rack change moves only the chassis. The scanner moves each
// module when it finds the module on the new rack.
type Writer struct {
	tx       gorp.Tx
	status   status.Writer
	statuses *status.Service
	table    *gorp.Table[Key, Chassis]
}

func (w Writer) setStatus(ctx context.Context, c Chassis) error {
	var s Status
	err := w.statuses.NewRetrieve[StatusDetails]().
		Where(status.MatchOwners[StatusDetails](c.ResourceID())).
		Entry(&s).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		s = Status{
			Owner:   new(c.ResourceID()),
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Status unknown",
			Details: StatusDetails{Chassis: c.Key},
		}
	} else if err != nil {
		return err
	}
	s.Name = c.Name
	s.Details.Rack = c.Rack
	return w.status.Set(ctx, &s)
}

// Create stores c and writes an unknown status when it has none. An existing status
// takes the name and rack of c. A chassis with a nil key gets a new one. It returns a
// validation error when c is invalid or its rack does not exist.
func (w Writer) Create(ctx context.Context, c *Chassis) error {
	if c.Key == uuid.Nil() {
		c.Key = uuid.New()
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(c).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.setStatus(ctx, *c)
}

// Update applies change to the chassis stored under key and stores the result. A
// rename or rack move updates the status, or writes it as unknown when it is missing.
// It returns an error wrapping query.ErrNotFound when no chassis is stored under key
// and a validation error when the result is invalid. A transaction that commits
// between the read and the write of this one is overwritten.
func (w Writer) Update(
	ctx context.Context,
	key Key,
	change func(Chassis) (Chassis, error),
) error {
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
		return w.setStatus(ctx, next)
	}
	return nil
}

// Delete removes the chassis stored under keys with their modules and statuses. Delete
// is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Chassis](keys...)).
		Exec(ctx, w.tx)
}
