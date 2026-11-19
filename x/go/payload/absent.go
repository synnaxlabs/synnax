// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload

import (
	"encoding/json/jsontext"

	"github.com/synnaxlabs/x/telem"
)

// Absent reports whether raw, the value at the pointer of a field, is no value for a
// channel of data type dt: missing, null except on a JSON channel, or an empty string
// on a channel that cannot hold one. A string, bytes, or JSON channel holds one.
func Absent(raw jsontext.Value, dt telem.DataType) bool {
	switch {
	case raw == nil:
		return true
	case dt == telem.JSONT:
		return false
	case raw.Kind() == 'n':
		return true
	case dt == telem.StringT, dt == telem.BytesT:
		return false
	}
	return string(raw) == `""`
}
