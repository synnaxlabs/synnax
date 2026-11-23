// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package broker

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Writer creates, updates, and deletes MQTT brokers in the transaction given to
// Service.NewWriter.
type Writer struct {
	tx       gorp.Tx
	status   status.Writer
	statuses *status.Service
	certs    *certificate.Service
	racks    *rack.Service
	table    *gorp.Table[Key, Broker]
}

// prepare validates b and refuses a rack that does not run MQTT and a client
// certificate with no private key.
func (w Writer) prepare(ctx context.Context, b Broker) error {
	if err := validateFields(&b); err != nil {
		return err
	}
	if err := w.checkRack(ctx, b); err != nil {
		return err
	}
	return w.checkClientCertificate(ctx, b)
}

func (w Writer) setStatus(ctx context.Context, b Broker) error {
	var st Status
	err := w.statuses.NewRetrieve[StatusDetails]().
		Where(status.MatchOwners[StatusDetails](b.ResourceID())).
		Entry(&st).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		st = Status{
			Owner:   new(b.ResourceID()),
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Status unknown",
			Details: StatusDetails{Broker: b.Key},
		}
	} else if err != nil {
		return err
	}
	st.Name = b.Name
	st.Details.Rack = b.Rack
	return w.status.Set(ctx, &st)
}

// Create stores b and writes an unknown status when it has none. An existing status
// takes the name and rack of b. A broker with a nil key gets a new one. It returns a
// validation error when b is invalid, holds a duplicate binding, keeps alive for a
// fraction of a second, or names a rack that does not run MQTT, a client certificate
// with no private key, or a file certificate or secret of another rack.
func (w Writer) Create(ctx context.Context, b *Broker) error {
	if b.Key == uuid.Nil() {
		b.Key = uuid.New()
	}
	if err := w.prepare(ctx, *b); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(b).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.setStatus(ctx, *b)
}

// update applies change to the broker stored under key. A rename or rack move updates
// the status, or writes it as unknown when it is missing.
func (w Writer) update(
	ctx context.Context,
	key Key,
	change func(Broker) (Broker, error),
) error {
	var prev, next Broker
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Broker](key)).
		ChangeErr(func(_ gorp.Context, b Broker) (Broker, error) {
			prev = b
			updated, err := change(b)
			if err != nil {
				return Broker{}, err
			}
			updated.Key = key
			if err = w.prepare(ctx, updated); err != nil {
				return Broker{}, err
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

// Delete removes the brokers stored under keys with their statuses. Delete is
// idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Broker](keys...)).
		Exec(ctx, w.tx)
}
