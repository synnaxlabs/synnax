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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/gorp"
)

// ChannelResolver looks up cluster channels by name or numeric key. It is
// the production dynamic resolver attached to a program root's
// GlobalResolver: cluster channels can appear or disappear at runtime,
// so we cannot snapshot them into the ambient prelude at analysis time.
type ChannelResolver struct {
	channelSvc *channel.Service
	tx         gorp.Tx
}

var _ symbol.Resolver = (*ChannelResolver)(nil)

func channelToSymbol(ch channel.Channel) *symbol.Symbol {
	return &symbol.Symbol{
		Name: ch.Name,
		Kind: symbol.KindChannel,
		Type: types.Chan(types.FromTelem(ch.DataType)),
		ID:   int(ch.Key()),
	}
}

func (r *ChannelResolver) Resolve(ctx context.Context, name string) (*symbol.Symbol, error) {
	key, err := strconv.Atoi(name)
	ch := channel.Channel{}
	q := r.channelSvc.NewRetrieve().Entry(&ch)
	if err == nil {
		q = q.Where(channel.MatchKeys(channel.Key(key)))
	} else {
		q = q.Where(channel.MatchNames(name))
	}
	if err = q.Exec(ctx, r.tx); err != nil {
		return nil, err
	}
	return channelToSymbol(ch), nil
}

func (r *ChannelResolver) Search(ctx context.Context, name string) ([]*symbol.Symbol, error) {
	var results []channel.Channel
	if err := r.channelSvc.NewRetrieve().
		Where(channel.MatchInternal(false)).
		Search(name).
		Entries(&results).Exec(ctx, r.tx); err != nil {
		return nil, err
	}
	return lo.Map(results, func(item channel.Channel, index int) *symbol.Symbol {
		return channelToSymbol(item)
	}), nil
}

// NewChannelResolver constructs the production dynamic resolver that
// reaches cluster channels by name or numeric key. The returned value
// satisfies symbol.Resolver via Go's interface implementation rules.
func NewChannelResolver(channelSvc *channel.Service, tx gorp.Tx) *ChannelResolver {
	return &ChannelResolver{channelSvc: channelSvc, tx: tx}
}
