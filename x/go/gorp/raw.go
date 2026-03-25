// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import "encoding/binary"

// SkipRawFields advances past n length-prefixed fields in binary-encoded data.
// Each field is encoded as [uint32 big-endian length][bytes]. Returns the remaining
// data after skipping, or nil if the data is too short.
func SkipRawFields(data []byte, n int) []byte {
	for range n {
		if len(data) < 4 {
			return nil
		}
		fieldLen := int(binary.BigEndian.Uint32(data[:4]))
		data = data[4:]
		if len(data) < fieldLen {
			return nil
		}
		data = data[fieldLen:]
	}
	return data
}

// ReadRawField reads one length-prefixed field from binary-encoded data. Returns
// the field value and the remaining data, or (nil, nil) if the data is too short.
func ReadRawField(data []byte) (value []byte, rest []byte) {
	if len(data) < 4 {
		return nil, nil
	}
	fieldLen := int(binary.BigEndian.Uint32(data[:4]))
	data = data[4:]
	if len(data) < fieldLen {
		return nil, nil
	}
	return data[:fieldLen], data[fieldLen:]
}
