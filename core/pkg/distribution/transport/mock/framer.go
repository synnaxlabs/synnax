// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/address"
)

type FramerNetwork struct {
	Iterator *FramerIteratorNetwork
	Writer   *FramerWriterNetwork
	Relay    *FramerRelayNetwork
	Deleter  *FramerDeleterNetwork
}

func NewFramerNetwork() *FramerNetwork {
	return &FramerNetwork{
		Iterator: NewIteratorNetwork(),
		Writer:   NewWriterNetwork(),
		Relay:    NewRelayNetwork(),
		Deleter:  NewDeleterNetwork(),
	}
}

func (f *FramerNetwork) New(add address.Address) framer.Transport {
	return &FramerTransport{
		iterator: f.Iterator.New(add),
		writer:   f.Writer.New(add),
		relay:    f.Relay.New(add),
	}
}

type FramerTransport struct {
	iterator iterator.Transport
	writer   writer.Transport
	relay    relay.Transport
	deleter  deleter.Transport
}

var (
	_ framer.Transport = (*FramerTransport)(nil)
)

func (c FramerTransport) Iterator() iterator.Transport { return c.iterator }

func (c FramerTransport) Writer() writer.Transport { return c.writer }

func (c FramerTransport) Relay() relay.Transport { return c.relay }

func (c FramerTransport) Deleter() deleter.Transport { return c.deleter }
