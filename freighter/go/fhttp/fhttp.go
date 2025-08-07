// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fhttp

import (
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/encoding/csv"
)

const (
	MIMEApplicationJSON    = fiber.MIMEApplicationJSON
	MIMEApplicationMsgPack = "application/msgpack"
	MIMETextCSV            = "text/csv"
	MIMETextPlain          = fiber.MIMETextPlain
)

type BindableTransport interface {
	freighter.Transport
	BindTo(*fiber.App)
}

type (
	// Encoder represents a codec that can encode values to an HTTP request / response
	// body.
	Encoder interface {
		binary.Encoder
		// ContentType returns the MIME type of the encoder.
		ContentType() string
	}
	// Decoder represents a codec that can decode values from an HTTP request / response
	// body.
	Decoder interface {
		binary.Decoder
		// ContentType returns the MIME type of the decoder.
		ContentType() string
	}
	// Codec represents a codec that can encode and decode values to and from an HTTP
	// request / response body.
	Codec interface {
		Encoder
		Decoder
	}
)

var (
	// JSONCodec is a codec that can encode and decode values to and from an HTTP
	// request / response body using the JSON format.
	JSONCodec = NewCodec(binary.JSONCodec, MIMEApplicationJSON)
	// MsgPackCodec is a codec that can encode and decode values to and from an HTTP
	// request / response body using the MessagePack format.
	MsgPackCodec = NewCodec(binary.MsgPackCodec, MIMEApplicationMsgPack)
	// TextCodec is a codec that can encode and decode values to and from an HTTP
	// request / response body using plain text.
	TextCodec = NewCodec(binary.StringCodec, MIMETextPlain)
	// CSVEncoder is an encoder that encodes values to an HTTP request / response body
	// using the CSV format.
	CSVEncoder = NewEncoder(csv.Encoder, MIMETextCSV)
)

type (
	typedDecoder struct {
		binary.Decoder
		contentType string
	}
	typedEncoder struct {
		binary.Encoder
		contentType string
	}
	typedCodec struct {
		binary.Codec
		contentType string
	}
)

func (c *typedCodec) ContentType() string   { return c.contentType }
func (d *typedDecoder) ContentType() string { return d.contentType }
func (e *typedEncoder) ContentType() string { return e.contentType }

// NewCodec creates a new fhttp codec from the given binary codec and content type.
func NewCodec(codec binary.Codec, contentType string) Codec {
	return &typedCodec{Codec: codec, contentType: contentType}
}

// NewDecoder creates a new fhttp decoder from the given content type and binary
// decoder.
func NewDecoder(decoder binary.Decoder, contentType string) Decoder {
	return &typedDecoder{Decoder: decoder, contentType: contentType}
}

// NewEncoder creates a new fhttp encoder from the given content type and binary
// encoder.
func NewEncoder(encoder binary.Encoder, contentType string) Encoder {
	return &typedEncoder{Encoder: encoder, contentType: contentType}
}

func newReporter(encodings ...string) freighter.Reporter {
	return freighter.Reporter{Encodings: lo.Uniq(encodings), Protocol: "http"}
}
