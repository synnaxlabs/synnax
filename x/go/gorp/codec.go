// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"
	"encoding/binary"
)

// Codec defines a custom encoding/decoding strategy for entries stored in a Table.
// When a Codec is set on a Table, it takes precedence over the default DB codec
// for value encoding/decoding.
type Codec[E any] interface {
	Marshal(ctx context.Context, entry E) ([]byte, error)
	Unmarshal(ctx context.Context, data []byte) (E, error)
}

// SkipRawFields advances past n length-prefixed (uint32 big-endian + bytes) fields,
// returning the remaining data. Returns nil if the data is malformed.
func SkipRawFields(data []byte, n int) []byte {
	for i := 0; i < n; i++ {
		if len(data) < 4 {
			return nil
		}
		fieldLen := binary.BigEndian.Uint32(data[:4])
		data = data[4:]
		if uint32(len(data)) < fieldLen {
			return nil
		}
		data = data[fieldLen:]
	}
	return data
}

// ReadRawField reads one length-prefixed field (uint32 big-endian length + bytes),
// returning the field bytes and the remaining data. Returns nil, nil if the data
// is malformed.
func ReadRawField(data []byte) (field, rest []byte) {
	if len(data) < 4 {
		return nil, nil
	}
	fieldLen := binary.BigEndian.Uint32(data[:4])
	data = data[4:]
	if uint32(len(data)) < fieldLen {
		return nil, nil
	}
	return data[:fieldLen], data[fieldLen:]
}
