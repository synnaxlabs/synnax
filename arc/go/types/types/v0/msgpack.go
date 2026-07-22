// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/vmihailenco/msgpack/v5"

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (c *Channels) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Channels
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(c)); err != nil {
		return err
	}
	if c.Read == nil && c.Write == nil {
		var legacy struct {
			Read  map[uint32]string
			Write map[uint32]string
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		c.Read = legacy.Read
		c.Write = legacy.Write
	}
	return nil
}
