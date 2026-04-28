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
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Codec pairs an encoding.Codec with the HTTP content type that identifies it over the
// wire. A Codec satisfies both RequestDecoder and ResponseEncoder, so any registered
// Codec can be used on either side of a request.
type Codec interface {
	RequestDecoder
	ResponseEncoder
}

// RequestDecoder is an encoding.Decoder paired with the HTTP content type it accepts on
// a request body. It is used by the unary HTTP server to select a decoder based on the
// request's Content-Type header.
type RequestDecoder interface {
	encoding.Decoder
	ContentType() string
}

// ResponseEncoder is an encoding.Encoder paired with the HTTP content type it produces
// on a response body. It is used by the unary HTTP server to select an encoder based
// on the request's Accept header.
type ResponseEncoder interface {
	encoding.Encoder
	ContentType() string
}

var codecs = []Codec{json.Codec, msgpack.Codec}

// ResolveCodec returns the registered Codec whose ContentType exactly matches the given
// content-type string, or an error if none match. Comparison is case-sensitive and does
// not strip parameters such as charset.
func ResolveCodec(contentType string) (Codec, error) {
	for _, codec := range codecs {
		if codec.ContentType() == contentType {
			return codec, nil
		}
	}
	return nil, errors.Newf(
		"[encoding] - unable to determine encoding type for %s",
		contentType,
	)
}

// SupportedContentTypes returns the content types of every registered Codec, in
// registration order. Useful for reporting and content negotiation.
func SupportedContentTypes() []string {
	var contentTypes []string
	for _, codec := range codecs {
		contentTypes = append(contentTypes, codec.ContentType())
	}
	return contentTypes
}

// DefaultRequestDecoders returns the registered codecs as a slice of RequestDecoders,
// suitable for seeding a unary server's decoder registry.
func DefaultRequestDecoders() []RequestDecoder {
	out := make([]RequestDecoder, len(codecs))
	for i, c := range codecs {
		out[i] = c
	}
	return out
}

// DefaultResponseEncoders returns the registered codecs as a slice of ResponseEncoders,
// suitable for seeding a unary server's encoder registry.
func DefaultResponseEncoders() []ResponseEncoder {
	out := make([]ResponseEncoder, len(codecs))
	for i, c := range codecs {
		out[i] = c
	}
	return out
}
