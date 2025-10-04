// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stage

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/x/signal"
)

type OutputHandler = func(ctx context.Context, param string, val value.Value)

type Stage interface {
	Key() string
	WriteChannels() []channel.Key
	ReadChannels() []channel.Key
	Flow(signal.Context)
	Next(ctx context.Context, param string, value value.Value)
	OnOutput(OutputHandler)
}
