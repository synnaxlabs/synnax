// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/telem"
)

type (
	Frame          = core.Frame
	Iterator       = iterator.Iterator
	StreamIterator = iterator.StreamIterator
	Writer         = writer.Writer
	StreamWriter   = writer.StreamWriter
	WriterConfig   = writer.Config
	IteratorConfig = iterator.Config
)

func NewFrame(keys channel.Keys, arrays []telem.Array) Frame {
	return core.NewFrame(keys, arrays)
}
