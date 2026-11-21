// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package master

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, updates, and deletes EtherCAT masters in the transaction given to
// Service.NewWriter. A rack change moves only the master. The scanner moves each slave
// when it finds the slave on the new rack.
type Writer struct {
	tx       gorp.Tx
	status   status.Writer
	statuses *status.Service
	table    *gorp.Table[Key, Master]
}

func (w Writer) setStatus(ctx context.Context, m Master) error {
	var s Status
	err := w.statuses.NewRetrieve[StatusDetails]().
		Where(status.MatchOwners[StatusDetails](m.ResourceID())).
		Entry(&s).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		s = Status{
			Owner:   new(m.ResourceID()),
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Status unknown",
			Details: StatusDetails{Master: m.Key},
		}
	} else if err != nil {
		return err
	}
	s.Name = m.Name
	s.Details.Rack = m.Rack
	return w.status.Set(ctx, &s)
}

func validateMaster(m Master) error {
	if err := m.Validate(); err != nil {
		return err
	}
	if len(lo.Uniq(m.Slaves)) != len(m.Slaves) {
		return errors.Wrapf(validate.ErrValidation, "slaves: must not repeat a key")
	}
	return nil
}

// Create stores m and writes an unknown status when it has none. An existing status
// takes the name and rack of m. A master with a nil key gets a new one. It returns a
// validation error when m is invalid, repeats a slave key, or its rack does not exist.
func (w Writer) Create(ctx context.Context, m *Master) error {
	if m.Key == uuid.Nil() {
		m.Key = uuid.New()
	}
	if err := validateMaster(*m); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(m).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.setStatus(ctx, *m)
}

// Update applies change to the master stored under key, inside the transaction, and
// stores the result. A rename or rack move updates the status, or writes it as unknown
// when it is missing. It returns an error wrapping query.ErrNotFound when no master is
// stored under key and a validation error when the result is invalid or repeats a
// slave key.
func (w Writer) Update(
	ctx context.Context,
	key Key,
	change func(Master) (Master, error),
) error {
	var prev, next Master
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Master](key)).
		ChangeErr(func(_ gorp.Context, m Master) (Master, error) {
			prev = m
			updated, err := change(m)
			if err != nil {
				return Master{}, err
			}
			updated.Key = key
			if err = validateMaster(updated); err != nil {
				return Master{}, err
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

// Delete removes the masters stored under keys with their slaves and statuses. Delete
// is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Master](keys...)).
		Exec(ctx, w.tx)
}
