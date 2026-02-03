// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
)

type (
	Frame            = frame.Frame
	Iterator         = iterator.Iterator
	IteratorRequest  = iterator.Request
	IteratorResponse = iterator.Response
	StreamIterator   = iterator.StreamIterator
	Writer           = writer.Writer
	WriterRequest    = writer.Request
	WriterResponse   = writer.Response
	StreamWriter     = writer.StreamWriter
	WriterConfig     = writer.Config
	IteratorConfig   = iterator.Config
	StreamerResponse = relay.Response
	StreamerRequest  = relay.Request
	StreamerConfig   = relay.StreamerConfig
	Streamer         = relay.Streamer
	Deleter          = deleter.Deleter
)
