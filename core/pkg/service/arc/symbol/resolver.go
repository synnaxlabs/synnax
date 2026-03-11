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
)

type channelResolver struct{ *channel.Service }

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
	q := r.NewRetrieve().Entry(&ch)
	if err == nil {
		q = q.WhereKeys(channel.Key(key))
	} else {
		q = q.WhereNames(name)
	}
	if err = q.Exec(ctx, nil); err != nil {
		return arc.Symbol{}, err
	}
	return channelToSymbol(ch), nil
}

func (r *channelResolver) Search(ctx context.Context, name string) ([]arc.Symbol, error) {
	var results []channel.Channel
	if err := r.NewRetrieve().
		WhereInternal(false).
		Search(name).
		Entries(&results).Exec(ctx, nil); err != nil {
		return nil, err
	}
	return lo.Map(results, func(item channel.Channel, index int) arc.Symbol {
		return channelToSymbol(item)
	}), nil
}

// DefaultSymbolResolver returns the default set of STL symbol resolvers used by
// the analyzer and LSP.
func DefaultSymbolResolver() symbol.CompoundResolver {
	resolvers := make(symbol.CompoundResolver, len(stl.SymbolResolver))
	copy(resolvers, stl.SymbolResolver)
	return resolvers
}

func CreateResolver(channelSvc *channel.Service, extraResolvers ...symbol.Resolver) arc.SymbolResolver {
	resolvers := DefaultSymbolResolver()
	resolvers = append(resolvers, arcstatus.SymbolResolver, &channelResolver{Service: channelSvc})
	resolvers = append(resolvers, extraResolvers...)
	return resolvers
}
