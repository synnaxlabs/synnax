// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

// Arrow returns the Arc operator for the kind: "->" for continuous, "=>" for
// conditional, and "?>" for any other kind.
func (k EdgeKind) Arrow() string {
	switch k {
	case EdgeKindContinuous:
		return "->"
	case EdgeKindConditional:
		return "=>"
	default:
		return "?>"
	}
}

// String returns the string representation of the edge. Format: "source.param ->
// target.param (continuous)" or "source.param => target.param (conditional)"
func (e Edge) String() string {
	return fmt.Sprintf("%s %s %s (%s)", e.Source, e.Kind.Arrow(), e.Target, e.Kind)
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase Go
// field names and new lowercase msgpack tag names for backward compatibility.
func (e *Edge) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Edge
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(e)); err != nil {
		return err
	}
	if len(e.Source.Node) == 0 {
		var legacy struct {
			Source Handle
			Target Handle
			Kind   EdgeKind
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		e.Source = legacy.Source
		e.Target = legacy.Target
		e.Kind = legacy.Kind
	}
	return nil
}
