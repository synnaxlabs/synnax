// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// Encoder is an encoding.Encoder paired with the HTTP content type it produces. Used by
// the unary HTTP server to encode a response body based on the Accept header, and by
// the unary HTTP client to encode an outgoing request body.
type Encoder interface {
	encoding.Encoder
	ContentType() string
}

// Decoder is an encoding.Decoder paired with the HTTP content type it accepts. Used by
// the unary HTTP server and client to decode a request and response body, based off of
// the Content-Type header.
type Decoder interface {
	encoding.Decoder
	ContentType() string
}

// Codec is both an Encoder and a Decoder for the same content type. The stream
// transport uses one Codec for both directions of a Websocket connection; the unary
// transport uses Codec values as defaults for both Encoder and Decoder registries.
type Codec interface {
	Encoder
	Decoder
}

// codecWithContentType pairs a content-type-agnostic encoding.Codec with the HTTP
// content type it represents. It lets the Freighter HTTP transport own the
// HTTP-specific ContentType association without leaking it into the underlying
// encoding packages, which are used in non-HTTP contexts (Pebble, file IO, etc.).
type codecWithContentType struct {
	encoding.Codec
	contentType string
}

func (c codecWithContentType) ContentType() string { return c.contentType }

// JSONCodec is the JSON Codec used by default for unary HTTP requests and responses,
// and as one of the default stream-server codecs. It wraps json.Codec from the
// x/encoding package with the application/json content type.
var JSONCodec Codec = codecWithContentType{
	Codec:       json.Codec,
	contentType: "application/json",
}

// MsgPackCodec is the MessagePack Codec used by default for unary HTTP requests and
// responses, the default unary client encoder, and the default stream client codec. It
// wraps msgpack.Codec from the x/encoding package with the application/msgpack content
// type.
var MsgPackCodec Codec = codecWithContentType{
	Codec:       msgpack.Codec,
	contentType: "application/msgpack",
}

var codecs = []Codec{JSONCodec, MsgPackCodec}

// defaultDecoders is the registered codecs as Decoders, used to seed a unary server's
// decoder registry. Read-only after init; callers replace the slice rather than
// mutating it.
var defaultDecoders = func() []Decoder {
	return lo.Map(codecs, func(c Codec, _ int) Decoder { return c })
}()

// defaultEncoders is the registered codecs as Encoders, used to seed a unary server's
// encoder registry. Read-only after init; callers replace the slice rather than
// mutating it.
var defaultEncoders = func() []Encoder {
	return lo.Map(codecs, func(c Codec, _ int) Encoder { return c })
}()
