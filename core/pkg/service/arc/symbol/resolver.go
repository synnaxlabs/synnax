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
	"github.com/synnaxlabs/arc/stl/authority"
	stlchannel "github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/stl/constant"
	"github.com/synnaxlabs/arc/stl/errors"
	"github.com/synnaxlabs/arc/stl/math"
	stlop "github.com/synnaxlabs/arc/stl/op"
	"github.com/synnaxlabs/arc/stl/selector"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stable"
	"github.com/synnaxlabs/arc/stl/stage"
	"github.com/synnaxlabs/arc/stl/stat"
	"github.com/synnaxlabs/arc/stl/stateful"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/telem"
	"github.com/synnaxlabs/arc/stl/time"
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

// DefaultResolverModules returns the default set of STL modules used for symbol
// resolution by the analyzer and LSP. These modules only need static symbol
// definitions and do not require runtime state.
// DefaultResolverModules returns the default set of STL modules used for symbol
// resolution by the analyzer and LSP. These modules only need static symbol
// definitions and do not require runtime state.
func DefaultResolverModules() []stl.Module {
	return []stl.Module{
		stlchannel.NewModule(nil, nil),
		stateful.NewModule(nil, nil),
		series.NewModule(nil),
		strings.NewModule(nil),
		math.NewModule(),
		errors.NewModule(),
		constant.NewModule(),
		stlop.NewModule(),
		selector.NewModule(),
		stable.NewModule(),
		authority.NewModule(nil),
		telem.NewModule(),
		stat.NewModule(),
		time.NewModule(),
		stage.NewModule(),
	}
}

func CreateResolver(channelSvc *channel.Service, modules ...stl.Module) arc.SymbolResolver {
	if len(modules) == 0 {
		modules = DefaultResolverModules()
	}
	resolvers := stl.CompoundResolver(modules...)
	return append(resolvers, arcstatus.SymbolResolver, &channelResolver{Service: channelSvc})
}
