// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter

import (
	"context"
	"io"

	"github.com/synnaxlabs/x/address"
)

// PipelineClient is the client-side interface for transports that support streaming
// response bodies within a single request-response cycle. This differs from
// StreamClient which handles bidirectional message exchange over time.
type PipelineClient[RQ Payload] interface {
	Transport
	// StreamResponse sends a complete request and streams the response body. The
	// response body is returned as an io.ReadCloser that must be closed by the caller.
	// This is useful for scenarios like file downloads or data exports.
	StreamResponse(context.Context, address.Address, RQ) (io.ReadCloser, error)
}

// PipelineServer is the server-side interface for transports that support streaming
// response bodies within a single request-response cycle.
type PipelineServer[RQ Payload] interface {
	Transport
	// BindHandler binds a handler that processes a complete request and returns a
	// streaming response body. The handler should return an io.Reader that will be
	// consumed to stream the response back to the client.
	BindHandler(func(context.Context, RQ) (io.Reader, error))
}
