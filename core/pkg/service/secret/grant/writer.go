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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates and deletes grants in one transaction. It makes no access check and
// does not check that the secrets exist: the caller does both.
type Writer struct {
	svc *Service
	tx  gorp.Tx
}

// Create stores g under a new key, which it writes to g. It returns a validation error
// when g's rack is not a rack, g names no secrets, or g has no session.
func (w Writer) Create(ctx context.Context, g *Grant) error {
	v := validate.New("grant")
	v.Ternaryf(
		"rack",
		g.Rack.Type != ontology.ResourceTypeRack,
		"%s is not a rack",
		g.Rack,
	)
	v.NotEmptySlice("secrets", g.Secrets)
	v.Ternary("session", g.Session == uuid.Nil(), "required")
	if err := v.Error(); err != nil {
		return err
	}
	g.Key = uuid.New()
	return w.svc.table.NewCreate().Entry(g).Exec(ctx, w.tx)
}

// Delete deletes the grants with the given keys. Unknown keys are ignored.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.svc.table.NewDelete().
		Where(gorp.MatchKeys[Key, Grant](keys...)).
		Exec(ctx, w.tx)
}
