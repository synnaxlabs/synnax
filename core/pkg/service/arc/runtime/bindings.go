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

	"github.com/synnaxlabs/arc/compiler/runtime"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/telem"
)

func (r *Runtime) bind() {
	b := runtime.NewBindings()
	b.ChannelReadU8 = r.channelReadU8
}

func (r *Runtime) channelReadU8(_ context.Context, key uint32) uint8 {
	v, ok := r.mu.values[channel.Key(key)]
	if !ok || v.Len() == 0 || v.DataType != telem.Uint8T {
		return 0
	}
	return telem.ValueAt[uint8](v, -1)
}
