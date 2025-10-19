// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/runtime/constant"
	"github.com/synnaxlabs/arc/runtime/op"
	"github.com/synnaxlabs/arc/runtime/selector"
	"github.com/synnaxlabs/arc/runtime/stable"
	"github.com/synnaxlabs/arc/runtime/telem"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/x/config"
)

type channelResolver struct {
	channel.Readable
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

func (r *channelResolver) ResolvePrefix(ctx context.Context, name string) ([]arc.Symbol, error) {
	var results []channel.Channel
	if err := r.NewRetrieve().Search(name).Entries(&results).Exec(ctx, nil); err != nil {
		return nil, err
	}
	return lo.Map(results, func(item channel.Channel, index int) arc.Symbol {
		return channelToSymbol(item)
	}), nil
}

func CreateResolver(cfgs ...runtime.Config) (arc.SymbolResolver, error) {
	cfg, err := config.New(runtime.DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return symbol.CompoundResolver{
		constant.SymbolResolver,
		op.SymbolResolver,
		selector.SymbolResolver,
		stable.SymbolResolver,
		status.SymbolResolver,
		telem.SymbolResolver,
		&channelResolver{Readable: cfg.Channel},
	}, nil
}
