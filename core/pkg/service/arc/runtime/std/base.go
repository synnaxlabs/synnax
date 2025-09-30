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

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/x/signal"
)

type base struct {
	key           string
	outputHandler stage.OutputHandler
	readChannels  []channel.Key
	writeChannels []channel.Key
}

var _ stage.Stage = (*base)(nil)

func (b *base) Key() string { return b.key }

func (b *base) ReadChannels() []channel.Key { return b.readChannels }

func (b *base) WriteChannels() []channel.Key { return b.writeChannels }

func (b *base) Flow(signal.Context) {}

func (b *base) Next(context.Context, string, value.Value) {}

func (b *base) OnOutput(handler stage.OutputHandler) {
	b.outputHandler = handler
}
