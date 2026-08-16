// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package http defines the codec interfaces shared between the x/encoding family of
// packages and any HTTP transport that negotiates body codecs by content type.
package http

import "github.com/synnaxlabs/x/encoding"

// Encoder is an encoding.Encoder paired with the HTTP content type it produces. Used
// by HTTP servers to encode a response body based on the Accept header, and by HTTP
// clients to encode an outgoing request body.
type Encoder interface {
	encoding.Encoder
	ContentType() string
}

// Decoder is an encoding.Decoder paired with the HTTP content type it accepts. Used
// by HTTP servers and clients to decode a request and response body based on the
// Content-Type header.
type Decoder interface {
	encoding.Decoder
	ContentType() string
}

// Codec is both an Encoder and a Decoder for the same content type.
type Codec interface {
	Encoder
	Decoder
}

// FileCodec is a Codec that also names the files its output belongs in, so a caller
// writing a directory can name each file without knowing which codec holds it.
type FileCodec interface {
	Codec
	encoding.FileEncoder
}
