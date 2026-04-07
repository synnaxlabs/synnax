// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
	"github.com/vmihailenco/msgpack/v5"
)

var _ gorp.Entry[uuid.UUID] = Arc{}

// GorpKey implements gorp.Entry.
func (s Arc) GorpKey() uuid.UUID { return s.Key }

// SetOptions implements gorp.Entry.
func (s Arc) SetOptions() []any { return nil }

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field name "Running" and new lowercase msgpack tag "running" for backward
// compatibility.
func (s *StatusDetails) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias StatusDetails
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(s)); err != nil {
		return err
	}
	if !s.Running {
		var legacy struct{ Running bool }
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		s.Running = legacy.Running
	}
	return nil
}
