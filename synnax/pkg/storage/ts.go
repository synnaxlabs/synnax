// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage

import (
	"github.com/synnaxlabs/cesium"
)

type (
	TS                 = cesium.DB
	TSWriter           = cesium.Writer
	TSIterator         = cesium.Iterator
	TSStreamWriter     = cesium.StreamWriter
	TSStreamIterator   = cesium.StreamIterator
	TSWriteRequest     = cesium.WriteRequest
	TSWriteResponse    = cesium.WriteResponse
	TSIteratorRequest  = cesium.IteratorRequest
	TSIteratorResponse = cesium.IteratorResponse
	TSChannelManager   = cesium.ChannelManager
	WritableTS         = cesium.Writable
	StreamWritableTS   = cesium.StreamWritable
	ReadableTS         = cesium.Readable
	StreamIterableTS   = cesium.StreamIterable
	Channel            = cesium.Channel
	Frame              = cesium.Frame
	ChannelKey         uint16
	IteratorConfig     = cesium.IteratorConfig
	WriterConfig       = cesium.WriterConfig
)

const AutoSpan = cesium.AutoSpan
