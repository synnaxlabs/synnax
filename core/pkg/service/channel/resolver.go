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
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/gorp"
)

// channelResolver resolves cluster channels into Arc symbols by name or numeric
// key. It is the dynamic resolver attached to a program root's GlobalResolver:
// cluster channels can appear or disappear at runtime, so they cannot be
// snapshotted into the ambient prelude at analysis time.
type channelResolver struct {
	dist *distchannel.Service
	tx   gorp.Tx
}

var _ symbol.Resolver = (*channelResolver)(nil)

func channelToSymbol(ch distchannel.Channel) *symbol.Symbol {
	return &symbol.Symbol{
		Name:       ch.Name,
		Kind:       symbol.KindChannel,
		Type:       types.Chan(types.FromTelem(ch.DataType)),
		ID:         int(ch.Key()),
		Renameable: !ch.Internal,
	}
}

// Resolve resolves a single cluster channel by name or numeric key.
func (r *channelResolver) Resolve(ctx context.Context, name string) (*symbol.Symbol, error) {
	key, err := strconv.Atoi(name)
	ch := distchannel.Channel{}
	q := r.dist.NewRetrieve().Entry(&ch)
	if err == nil {
		q = q.Where(distchannel.MatchKeys(distchannel.Key(key)))
	} else {
		q = q.Where(distchannel.MatchNames(name))
	}
	if err = q.Exec(ctx, r.tx); err != nil {
		return nil, err
	}
	return channelToSymbol(ch), nil
}

// Search fuzzy-searches non-internal cluster channels by name.
func (r *channelResolver) Search(ctx context.Context, name string) ([]*symbol.Symbol, error) {
	var results []distchannel.Channel
	if err := r.dist.NewRetrieve().
		Where(distchannel.MatchInternal(false)).
		Search(name).
		Entries(&results).Exec(ctx, r.tx); err != nil {
		return nil, err
	}
	return lo.Map(results, func(item distchannel.Channel, _ int) *symbol.Symbol {
		return channelToSymbol(item)
	}), nil
}
