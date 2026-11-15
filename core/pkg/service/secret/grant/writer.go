// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grant

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates grants in one transaction. It makes no access check and does not check
// that the secrets exist: the caller does both.
type Writer struct {
	svc *Service
	tx  gorp.Tx
}

// Create stores g under a new key with an expiry one TTL from now, and writes both to
// g. It also deletes the grants that have expired. It returns a validation error when
// g's subject is not a rack or g names no secrets.
func (w Writer) Create(ctx context.Context, g *Grant) error {
	v := validate.New("grant")
	v.Ternaryf(
		"subject",
		g.Subject.Type != resource.TypeRack,
		"%s is not a rack",
		g.Subject,
	)
	v.NotEmptySlice("secrets", g.Secrets)
	if err := v.Error(); err != nil {
		return err
	}
	now := w.svc.now()
	if err := w.svc.table.NewDelete().
		Where(gorp.Match(func(_ gorp.Context, e *Grant) (bool, error) {
			return e.ExpiresAt.Before(now), nil
		})).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	g.Key = uuid.New()
	g.ExpiresAt = now.Add(telem.TimeSpan(w.svc.cfg.TTL))
	return w.svc.table.NewCreate().Exclusive().Entry(g).Exec(ctx, w.tx)
}
