// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/x/gorp"
)

type channelResolver struct {
	channelSvc *channel.Service
	tx         gorp.Tx
}

var _ arc.SymbolResolver = (*channelResolver)(nil)

func channelToSymbol(ch channel.Channel) symbol.Symbol {
	return arc.Symbol{
		Name: ch.Name,
		Kind: symbol.KindChannel,
		Type: types.Chan(types.FromTelem(ch.DataType)),
		ID:   int(ch.Key()),
	}
}

func (r *channelResolver) Resolve(ctx context.Context, name string) (arc.Symbol, error) {
	key, err := strconv.Atoi(name)
	ch := channel.Channel{}
	q := r.channelSvc.NewRetrieve().Entry(&ch)
	if err == nil {
		q = q.Where(channel.MatchKeys(channel.Key(key)))
	} else {
		q = q.Where(channel.MatchNames(name))
	}
	if err = q.Exec(ctx, r.tx); err != nil {
		return arc.Symbol{}, err
	}
	return channelToSymbol(ch), nil
}

func (r *channelResolver) Search(ctx context.Context, name string) ([]arc.Symbol, error) {
	var results []channel.Channel
	if err := r.channelSvc.NewRetrieve().
		Where(channel.MatchInternal(false)).
		Search(name).
		Entries(&results).Exec(ctx, r.tx); err != nil {
		return nil, err
	}
	return lo.Map(results, func(item channel.Channel, index int) arc.Symbol {
		return channelToSymbol(item)
	}), nil
}

func NewResolver(channelSvc *channel.Service, tx gorp.Tx) arc.SymbolResolver {
	resolvers := make(symbol.CompoundResolver, len(stl.SymbolResolver))
	copy(resolvers, stl.SymbolResolver)
	return append(
		resolvers,
		arcstatus.SymbolResolver,
		&channelResolver{channelSvc: channelSvc, tx: tx},
	)
}
