// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Writer creates, updates, and deletes Kafka clusters in the transaction given to
// Service.NewWriter.
type Writer struct {
	tx       gorp.Tx
	status   status.Writer
	statuses *status.Service
	racks    *rack.Service
	table    *gorp.Table[Key, Cluster]
}

// prepare validates c and refuses a rack that does not run Kafka.
func (w Writer) prepare(ctx context.Context, c Cluster) error {
	if err := validateFields(&c); err != nil {
		return err
	}
	return w.checkRack(ctx, c)
}

func (w Writer) setStatus(ctx context.Context, c Cluster) error {
	var st Status
	err := w.statuses.NewRetrieve[StatusDetails]().
		Where(status.MatchOwners[StatusDetails](c.ResourceID())).
		Entry(&st).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		st = Status{
			Owner:   new(c.ResourceID()),
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Status unknown",
			Details: StatusDetails{Cluster: c.Key},
		}
	} else if err != nil {
		return err
	}
	st.Name = c.Name
	st.Details.Rack = c.Rack
	return w.status.Set(ctx, &st)
}

// Create stores c and writes an unknown status when it has none. An existing status
// takes the name and rack of c. A cluster with a nil key gets a new one. It returns a
// validation error when c is invalid, has no brokers, holds a duplicate broker or
// read, or names a rack that does not run Kafka or a file certificate or secret of
// another rack.
func (w Writer) Create(ctx context.Context, c *Cluster) error {
	if c.Key == uuid.Nil() {
		c.Key = uuid.New()
	}
	if err := w.prepare(ctx, *c); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(c).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.setStatus(ctx, *c)
}

// update applies change to the cluster stored under key. A rename or rack move updates
// the status, or writes it as unknown when it is missing.
func (w Writer) update(
	ctx context.Context,
	key Key,
	change func(Cluster) (Cluster, error),
) error {
	var prev, next Cluster
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Cluster](key)).
		ChangeErr(func(_ gorp.Context, c Cluster) (Cluster, error) {
			prev = c
			updated, err := change(c)
			if err != nil {
				return Cluster{}, err
			}
			updated.Key = key
			if err = w.prepare(ctx, updated); err != nil {
				return Cluster{}, err
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

// Delete removes the clusters stored under keys with their statuses. Delete is
// idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Cluster](keys...)).
		Exec(ctx, w.tx)
}
