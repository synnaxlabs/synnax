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
	Frame = frame.Frame

	Iterator                = iterator.Iterator
	StreamIterator          = iterator.StreamIterator
	IteratorConfig          = iterator.Config
	IteratorRequest         = iterator.Request
	IteratorResponse        = iterator.Response
	IteratorCommand         = iterator.Command
	IteratorResponseVariant = iterator.ResponseVariant
	IteratorClient          = iterator.Client
	IteratorServer          = iterator.Server
	IteratorClientStream    = iterator.ClientStream
	IteratorServerStream    = iterator.ServerStream
	IteratorTransport       = iterator.Transport

	Writer             = writer.Writer
	StreamWriter       = writer.StreamWriter
	WriterConfig       = writer.Config
	WriterRequest      = writer.Request
	WriterResponse     = writer.Response
	WriterCommand      = writer.Command
	WriterMode         = writer.Mode
	WriterClient       = writer.Client
	WriterServer       = writer.Server
	WriterClientStream = writer.ClientStream
	WriterServerStream = writer.ServerStream
	WriterTransport    = writer.Transport

	Streamer             = relay.Streamer
	StreamerConfig       = relay.StreamerConfig
	StreamerRequest      = relay.Request
	StreamerResponse     = relay.Response
	StreamerClient       = relay.Client
	StreamerServer       = relay.Server
	StreamerClientStream = relay.ClientStream
	StreamerServerStream = relay.ServerStream
	StreamerTransport    = relay.Transport

	DeleterRequest   = deleter.Request
	DeleterClient    = deleter.Client
	DeleterServer    = deleter.Server
	DeleterTransport = deleter.Transport
)

const (
	IteratorCommandNext      = iterator.CommandNext
	IteratorCommandPrev      = iterator.CommandPrev
	IteratorCommandSeekFirst = iterator.CommandSeekFirst
	IteratorCommandSeekLast  = iterator.CommandSeekLast
	IteratorCommandSeekLE    = iterator.CommandSeekLE
	IteratorCommandSeekGE    = iterator.CommandSeekGE
	IteratorCommandValid     = iterator.CommandValid
	IteratorCommandError     = iterator.CommandError
	IteratorCommandSetBounds = iterator.CommandSetBounds

	IteratorResponseVariantAck  = iterator.ResponseVariantAck
	IteratorResponseVariantData = iterator.ResponseVariantData

	WriterCommandOpen         = writer.CommandOpen
	WriterCommandWrite        = writer.CommandWrite
	WriterCommandCommit       = writer.CommandCommit
	WriterCommandSetAuthority = writer.CommandSetAuthority
)

var (
	NewUnary       = frame.NewUnary
	NewMulti       = frame.NewMulti
	NewFromStorage = frame.NewFromStorage
)
