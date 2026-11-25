// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// UnmarshalJSONFrom decodes a read binding. An empty data type, which clients from
// before the data type was optional send for none, decodes as absent.
func (b *ReadBinding) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain ReadBinding
	var p plain
	if err := json.UnmarshalDecode(dec, &p); err != nil {
		return err
	}
	if p.DataType != nil && *p.DataType == "" {
		p.DataType = nil
	}
	*b = ReadBinding(p)
	return nil
}
