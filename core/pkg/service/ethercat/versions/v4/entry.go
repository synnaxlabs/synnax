// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// UnmarshalJSONFrom decodes an entry read channel. An empty data type, which clients
// from before the data type was optional send for none, decodes as absent.
func (e *EntryReadChannel) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain EntryReadChannel
	if err := json.UnmarshalDecode(dec, (*plain)(e)); err != nil {
		return err
	}
	if e.DataType != nil && *e.DataType == "" {
		e.DataType = nil
	}
	return nil
}

// UnmarshalJSONFrom decodes an entry write channel. An empty data type, which clients
// from before the data type was optional send for none, decodes as absent.
func (e *EntryWriteChannel) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain EntryWriteChannel
	if err := json.UnmarshalDecode(dec, (*plain)(e)); err != nil {
		return err
	}
	if e.DataType != nil && *e.DataType == "" {
		e.DataType = nil
	}
	return nil
}
