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

// Node is the core data processing structure of a reactive arc program. A stage
// performs a specific piece of functionality (such as an addition or accumulation),
// and then emits a value that can be used to trigger a downstream stage.
//
// An instantiation of a stage is called a node.
type Node interface {
	// Key is a unique key identifying the instantiated stage within the arc program.
	Key() string
	// WriteChannels are the keys of the channels that the stage needs to write to
	// in order to operate.
	WriteChannels() []channel.Key
	// ReadChannels are the keys of the channels that the stage needs to read from
	// in order to operate.
	ReadChannels() []channel.Key
	// Flow is used to start any go-routines that the stage may need.
	Flow(signal.Context)
	// Load loads the next value with the target param n into the stage. This parameter
	// should be used in the next evaluation through Next.
	Load(param string, value value.Value)
	// OnOutput binds an output handler that will be called when one of the output
	// parameters of the stage changes.
	OnOutput(OutputHandler)
	// Next executes the next 'step' in the stage to read parameters and conditionally
	// call the output handler.
	Next(ctx context.Context)
}
