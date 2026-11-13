// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload

import "github.com/synnaxlabs/x/telem"

// KindOf returns the kind of a read field that writes to a channel of data type dt. ok
// is false when dt is not a known data type.
func KindOf(dt telem.DataType) (k Kind, ok bool) {
	switch dt {
	case telem.Float32T, telem.Float64T:
		return KindFloat, true
	case telem.Int8T, telem.Int16T, telem.Int32T, telem.Int64T:
		return KindInteger, true
	case telem.Uint8T, telem.Uint16T, telem.Uint32T, telem.Uint64T:
		return KindUnsigned, true
	case telem.BooleanT:
		return KindBoolean, true
	case telem.StringT:
		return KindString, true
	case telem.TimestampT:
		return KindTimestamp, true
	case telem.JSONT:
		return KindJSON, true
	case telem.UUIDT:
		return KindUUID, true
	case telem.BytesT:
		return KindBytes, true
	default:
		return "", false
	}
}
