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

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

type channelResolver struct {
	channel.Readable
}

var _ arc.SymbolResolver = (*channelResolver)(nil)

func (r *channelResolver) Resolve(ctx context.Context, name string) (arc.Symbol, error) {
	c := r.NewRetrieve().WhereNames(name).Exec(ctx, nil)
}

func CreateResolver() {

}
