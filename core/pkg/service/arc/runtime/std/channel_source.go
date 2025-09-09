// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package std

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
)

type channelSource struct{ base }

func (c *channelSource) Next(ctx context.Context, value stage.Value) {
	c.outputHandler(ctx, value)
}

func createChannelSource(_ context.Context, node ir.Node) (stage.Stage, error) {
	ch := node.Config["channel"].(channel.Key)
	source := &channelSource{}
	source.readChannels = []channel.Key{ch}
	return source, nil
}
