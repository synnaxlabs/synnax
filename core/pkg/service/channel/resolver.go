// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/gorp"
)

// symbolResolver resolves cluster channels into Arc symbols by name or numeric key. It
// is the dynamic resolver attached to a program root's GlobalResolver: cluster channels
// can appear or disappear at runtime, so they cannot be snapshotted into the ambient
// prelude at analysis time.
type symbolResolver struct {
	svc *Service
	tx  gorp.Tx
}

var _ symbol.Resolver = (*symbolResolver)(nil)

func toSymbol(ch Channel) *symbol.Symbol {
	return &symbol.Symbol{
		Name:       ch.Name,
		Kind:       symbol.KindChannel,
		Type:       types.Chan(types.FromTelem(ch.DataType)),
		ID:         int(ch.Key()),
		Renameable: !ch.Internal,
	}
}

// Resolve resolves a single cluster channel by name or numeric key.
func (r *symbolResolver) Resolve(
	ctx context.Context,
	name string,
) (*symbol.Symbol, error) {
	key, err := strconv.Atoi(name)
	var ch Channel
	q := r.svc.NewRetrieve().Entry(&ch)
	if err == nil {
		q = q.Where(MatchKeys(Key(key)))
	} else {
		q = q.Where(MatchNames(name))
	}
	if err = q.Exec(ctx, r.tx); err != nil {
		return nil, err
	}
	return toSymbol(ch), nil
}

// Search fuzzy-searches non-internal cluster channels by name.
func (r *symbolResolver) Search(
	ctx context.Context,
	name string,
) ([]*symbol.Symbol, error) {
	var results []Channel
	if err := r.svc.NewRetrieve().
		Where(MatchInternal(false)).
		Search(name).
		Entries(&results).Exec(ctx, r.tx); err != nil {
		return nil, err
	}
	return lo.Map(results, func(item Channel, _ int) *symbol.Symbol {
		return toSymbol(item)
	}), nil
}
