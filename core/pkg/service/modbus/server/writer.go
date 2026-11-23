// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"context"
	"strconv"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// registerKey is a comparable form of a Register.
type registerKey struct {
	typ      RegisterType
	address  uint16
	dataType DataType
	bit      int
}

func keyOf(r Register) registerKey {
	k := registerKey{typ: r.Type, address: r.Address, bit: -1}
	if r.DataType != nil {
		k.dataType = *r.DataType
	}
	if r.Bit != nil {
		k.bit = int(*r.Bit)
	}
	return k
}

// validateServer checks s against its schema. It also checks that no two reads share a
// register and an index, and that no two writes share a register.
func validateServer(s Server) error {
	if err := s.Validate(); err != nil {
		return err
	}
	v := validate.New("Server")
	type readKey struct {
		register registerKey
		index    channel.Key
	}
	reads := set.New[readKey]()
	for i, r := range s.Reads {
		k := readKey{register: keyOf(r.Register), index: r.Index}
		v.Ternary(
			"reads."+strconv.Itoa(i),
			reads.Contains(k),
			"repeats the register and index of another read",
		)
		reads.Add(k)
	}
	writes := set.New[registerKey]()
	for i, w := range s.Writes {
		k := keyOf(w.Register)
		v.Ternary(
			"writes."+strconv.Itoa(i),
			writes.Contains(k),
			"repeats the register of another write",
		)
		writes.Add(k)
	}
	return v.Error()
}

// Writer creates, updates, and deletes Modbus servers in the transaction given to
// Service.NewWriter.
type Writer struct {
	tx       gorp.Tx
	status   status.Writer
	statuses *status.Service
	table    *gorp.Table[Key, Server]
}

func (w Writer) setStatus(ctx context.Context, s Server) error {
	var st Status
	err := w.statuses.NewRetrieve[StatusDetails]().
		Where(status.MatchOwners[StatusDetails](s.ResourceID())).
		Entry(&st).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		st = Status{
			Owner:   new(s.ResourceID()),
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Status unknown",
			Details: StatusDetails{Server: s.Key},
		}
	} else if err != nil {
		return err
	}
	st.Name = s.Name
	st.Details.Rack = s.Rack
	return w.status.Set(ctx, &st)
}

// Create stores s and writes an unknown status when it has none. An existing status
// takes the name and rack of s. A server with a nil key gets a new one. It returns a
// validation error when s is invalid or when its rack does not exist.
func (w Writer) Create(ctx context.Context, s *Server) error {
	if s.Key == uuid.Nil() {
		s.Key = uuid.New()
	}
	if err := validateServer(*s); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.setStatus(ctx, *s)
}

// update applies change to the server stored under key. A rename or rack move updates
// the status, or writes it as unknown when it is missing.
func (w Writer) update(
	ctx context.Context,
	key Key,
	change func(Server) (Server, error),
) error {
	var prev, next Server
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Server](key)).
		ChangeErr(func(_ gorp.Context, s Server) (Server, error) {
			prev = s
			updated, err := change(s)
			if err != nil {
				return Server{}, err
			}
			updated.Key = key
			if err = validateServer(updated); err != nil {
				return Server{}, err
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

// Delete removes the servers stored under keys with their statuses. Delete is
// idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Server](keys...)).
		Exec(ctx, w.tx)
}
