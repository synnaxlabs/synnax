// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"context"
	"strconv"

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/std"
	"github.com/synnaxlabs/x/config"
)

type channelResolver struct {
	channel.Readable
}

var _ arc.SymbolResolver = (*channelResolver)(nil)

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
	return arc.Symbol{
		Name: name,
		Kind: ir.KindChannel,
		Type: ir.Chan{ValueType: ir.TypeFromTelem(ch.DataType)},
		ID:   int(ch.Key()),
	}, nil
}

func CreateResolver(cfgs ...Config) (arc.SymbolResolver, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	r := ir.CompoundResolver{
		std.Resolver,
		&channelResolver{Readable: cfg.Channel},
	}
	return r, nil
}
