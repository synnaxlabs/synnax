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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
)

// Transport bundles the node-to-node transports for the framer's operations: streaming
// writes, streaming historical reads (iterator), streaming live reads (relay), and unary
// deletes.
type Transport interface {
	// Deleter returns the transport for deleting data from channels.
	Deleter() deleter.Transport
	// Iterator returns the transport for streaming historical reads.
	Iterator() iterator.Transport
	// Relay returns the transport for streaming live reads.
	Relay() relay.Transport
	// Writer returns the transport for streaming writes.
	Writer() writer.Transport
}
