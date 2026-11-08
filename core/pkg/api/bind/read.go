// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bind

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filter"
)

// Named is a retrieve request that names records by key.
type Named interface {
	// Named returns the records that the request names.
	Named() []resource.ID
}

// Held is a retrieve request that reads the API keys or sessions of users or racks.
type Held interface {
	// Holders returns the users and racks whose API keys or sessions the request reads.
	Holders() []resource.ID
}

// Keys returns the resource ID, made by id, of each key that set selects. A nil set
// selects none.
func Keys[K any](set *filter.Set[K], id func(K) resource.ID) []resource.ID {
	if set == nil {
		return nil
	}
	ids := make([]resource.ID, 0, len(set.In)+1)
	if set.Eq != nil {
		ids = append(ids, id(*set.Eq))
	}
	for _, k := range set.In {
		ids = append(ids, id(k))
	}
	return ids
}

// Snapshot is the committed state that a retrieve reads, with the read access of the
// subject of the request.
type Snapshot struct {
	gorp.Snapshot
	subject resource.ID
	allowed func(resource.ID) bool
}

// Allowed reports whether the subject of the request can retrieve the record with id.
func (s Snapshot) Allowed(id resource.ID) bool { return s.allowed(id) }

// Enforce returns an error that wraps access.ErrDenied and names the first of ids that
// the subject of the request cannot retrieve.
func (s Snapshot) Enforce(ids ...resource.ID) error {
	for _, id := range ids {
		if !s.allowed(id) {
			return errors.Wrapf(
				access.ErrDenied,
				"%s cannot %s %s",
				s.subject,
				access.ActionRetrieve,
				id,
			)
		}
	}
	return nil
}

// Filter returns a query filter that keeps the records that the subject of snap can
// retrieve. match is the Match function of the service that stores the records.
func Filter[R any, E interface{ ResourceID() resource.ID }, F any](
	snap Snapshot,
	match func(func(gorp.Context, R, *E) (bool, error)) F,
) F {
	return match(func(_ gorp.Context, _ R, e *E) (bool, error) {
		return snap.allowed((*e).ResourceID()), nil
	})
}

// Read returns a handler that runs retrieve against one snapshot of committed state. No
// write commits while retrieve runs, so retrieve must not write or open a transaction.
// The subject needs retrieve on each record that a Named request names, and on each
// holder of a Held request other than itself: as with Write, a subject needs no policy
// for its own API keys and sessions. Else the handler returns an error that wraps
// access.ErrDenied and names the record. retrieve leaves out the other records that the
// subject cannot retrieve, through Filter or Snapshot.Allowed. Policies are read once,
// from the snapshot. An error returns a zero response.
func Read[RQ, RS freighter.Payload](
	s *Service,
	retrieve func(context.Context, Snapshot, RQ) (RS, error),
) freighter.UnaryHandler[RQ, RS] {
	return func(ctx context.Context, req RQ) (RS, error) {
		var res, zero RS
		if err := s.cfg.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
			subject := auth.GetSubject(ctx)
			allowed, err := s.cfg.RBAC.NewEnforcer(snap).
				Allowed(ctx, subject, access.ActionRetrieve)
			if err != nil {
				return err
			}
			rs := Snapshot{Snapshot: snap, subject: subject, allowed: allowed}
			if n, ok := any(req).(Named); ok {
				if err = rs.Enforce(n.Named()...); err != nil {
					return err
				}
			}
			if h, ok := any(req).(Held); ok {
				if err = rs.Enforce(lo.Without(h.Holders(), subject)...); err != nil {
					return err
				}
			}
			res, err = retrieve(ctx, rs, req)
			return err
		}); err != nil {
			return zero, err
		}
		return res, nil
	}
}
