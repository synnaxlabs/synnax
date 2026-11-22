// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slave

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Writer creates, updates, and deletes EtherCAT slaves in the transaction given to
// Service.NewWriter.
type Writer struct {
	tx       gorp.Tx
	status   status.Writer
	statuses *status.Service
	table    *gorp.Table[Key, Slave]
}

func (w Writer) setStatus(ctx context.Context, sl Slave) error {
	var s Status
	err := w.statuses.NewRetrieve[StatusDetails]().
		Where(status.MatchOwners[StatusDetails](sl.ResourceID())).
		Entry(&s).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		s = Status{
			Owner:   new(sl.ResourceID()),
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Status unknown",
			Details: StatusDetails{Slave: sl.Key},
		}
	} else if err != nil {
		return err
	}
	s.Name = sl.Name
	s.Details.Rack = sl.Rack
	return w.status.Set(ctx, &s)
}

// Create stores s and writes an unknown status when it has none. An existing status
// takes the name and rack of s. A slave with a nil key gets a new one. It returns a
// validation error when s is invalid or its rack or master does not exist.
func (w Writer) Create(ctx context.Context, s *Slave) error {
	if s.Key == uuid.Nil() {
		s.Key = uuid.New()
	}
	if err := s.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.setStatus(ctx, *s)
}

// update applies change to the slave stored under key and writes the status on a
// rename or rack move.
func (w Writer) update(
	ctx context.Context,
	key Key,
	change func(Slave) (Slave, error),
) error {
	var prev, next Slave
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Slave](key)).
		ChangeErr(func(_ gorp.Context, s Slave) (Slave, error) {
			prev = s
			updated, err := change(s)
			if err != nil {
				return Slave{}, err
			}
			updated.Key = key
			if err = updated.Validate(); err != nil {
				return Slave{}, err
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

// Delete removes the slaves stored under keys with their statuses. Each master keeps
// the keys in its bus order until the next scan rewrites it. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Slave](keys...)).
		Exec(ctx, w.tx)
}
