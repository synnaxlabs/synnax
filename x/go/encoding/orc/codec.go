// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package orc

import (
	"context"
	"io"
	"sync"

	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// magic is the 3-byte header written at the start of every ORC-encoded payload.
// It allows quick format detection without trial decoding. The bytes spell "ORC"
// in ASCII and do not conflict with msgpack (0x80-0xdf, 0xc0-0xd3) or JSON
// (0x22-0x7b) leading bytes.
const magicLen = 3

var magic = [magicLen]byte{0x4F, 0x52, 0x43}

var ErrInvalidFormat = errors.Wrap(validate.ErrValidation, "data was not encoded using orc")

func validateMagic(data []byte) error {
	if len(data) < len(magic) || data[0] != magic[0] || data[1] != magic[1] || data[2] != magic[2] {
		return ErrInvalidFormat
	}
	return nil
}

// SelfEncoder is implemented by types that can encode themselves to ORC binary format.
type SelfEncoder interface {
	EncodeOrc(w *Writer) error
}

// SelfDecoder is implemented by types that can decode themselves from ORC binary format.
type SelfDecoder interface {
	DecodeOrc(r *Reader) error
}

// SelfCodec is implemented by types that can both encode and decode themselves
// using the ORC binary format.
type SelfCodec interface {
	SelfEncoder
	SelfDecoder
}

var (
	writerPool = sync.Pool{New: func() any { return NewWriter(0) }}
	readerPool = sync.Pool{New: func() any { return NewReader(nil) }}
)

// Codec is an orc implementation of encoding.Codec that requires all values to
// implement SelfEncoder/SelfDecoder.
var Codec = &codec{}

type codec struct {
	fallback encoding.Codec
}

// NewCodec returns an orc codec that falls back to the given codec when a value
// does not implement SelfEncoder (on encode) or when the data lacks the orc magic
// header (on decode).
func NewCodec(fallback encoding.Codec) encoding.Codec {
	return &codec{fallback: fallback}
}

func (c *codec) Encode(ctx context.Context, value any) ([]byte, error) {
	var m SelfEncoder
	if c.fallback != nil {
		var ok bool
		m, ok = value.(SelfEncoder)
		if !ok {
			return c.fallback.Encode(ctx, value)
		}
	} else {
		var ok bool
		m, ok = value.(SelfEncoder)
		if !ok {
			return nil, errors.Newf("orc: %T does not implement SelfEncoder", value)
		}
	}
	w := writerPool.Get().(*Writer)
	w.Reset()
	w.Write(magic[:])
	err := m.EncodeOrc(w)
	out := w.Copy()
	writerPool.Put(w)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *codec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	b, err := c.Encode(ctx, value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func (c *codec) Decode(ctx context.Context, data []byte, value any) error {
	if err := validateMagic(data); err != nil {
		if c.fallback != nil {
			return c.fallback.Decode(ctx, data, value)
		}
		return errors.New("orc: invalid magic header")
	}
	m, ok := value.(SelfDecoder)
	if !ok {
		return errors.Newf("orc: %T does not implement SelfDecoder", value)
	}
	r := readerPool.Get().(*Reader)
	r.ResetBytes(data[len(magic):])
	err := m.DecodeOrc(r)
	readerPool.Put(r)
	return err
}

func (c *codec) DecodeStream(ctx context.Context, rd io.Reader, value any) error {
	data, err := io.ReadAll(rd)
	if err != nil {
		return err
	}
	return c.Decode(ctx, data, value)
}
