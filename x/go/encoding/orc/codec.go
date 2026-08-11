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

// magic is the 3-byte header written at the start of every ORC-encoded payload. It
// allows quick format detection without trial decoding. The bytes spell "ORC" in ASCII
// and do not conflict with MessagePack (0x80-0xdf, 0xc0-0xd3) or JSON (0x22-0x7b)
// leading bytes.
var magic = [3]byte{0x4F, 0x52, 0x43}

func validateMagic(data []byte) error {
	if len(data) < len(magic) || data[0] != magic[0] || data[1] != magic[1] ||
		data[2] != magic[2] {
		return errors.Wrap(validate.ErrValidation, "data was not encoded using ORC")
	}
	return nil
}

// SelfEncoder is implemented by types that can encode themselves to ORC binary format.
type SelfEncoder interface {
	EncodeOrc(*Writer) error
}

// SelfDecoder is implemented by types that can decode themselves from ORC binary
// format.
type SelfDecoder interface {
	DecodeOrc(*Reader) error
}

// SelfCodec is implemented by types that can both encode and decode themselves using
// the ORC binary format.
type SelfCodec interface {
	SelfEncoder
	SelfDecoder
}

var (
	writerPool = sync.Pool{New: func() any { return NewWriter(0) }}
	readerPool = sync.Pool{New: func() any { return NewReader(nil) }}
)

// Codec is an Orc implementation of encoding.Codec that requires all values to
// implement SelfEncoder/SelfDecoder. Decode returns an error wrapping
// validate.ErrValidation for data without the Orc magic header; compose with
// encoding.NewDecodeFallbackCodec to read data written by another codec.
var Codec encoding.Codec = &codec{}

type codec struct{}

func (*codec) Decode(_ context.Context, data []byte, value any) error {
	if err := validateMagic(data); err != nil {
		return err
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

func (*codec) Encode(_ context.Context, value any) ([]byte, error) {
	m, ok := value.(SelfEncoder)
	if !ok {
		return nil, errors.Newf("orc: %T does not implement SelfEncoder", value)
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

func (c *codec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	m, ok := value.(SelfEncoder)
	if !ok {
		return errors.Newf("orc: %T does not implement SelfEncoder", value)
	}
	ow := writerPool.Get().(*Writer)
	ow.Reset()
	ow.Write(magic[:])
	err := m.EncodeOrc(ow)
	if err == nil {
		_, err = w.Write(ow.Bytes())
	}
	writerPool.Put(ow)
	return err
}
