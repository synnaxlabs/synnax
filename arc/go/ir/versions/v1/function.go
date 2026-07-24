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
	"strings"

	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/tree"
	"github.com/vmihailenco/msgpack/v5"
)

// Type returns the type signature of f.
func (f Function) Type() types.Type {
	return types.Function(types.FunctionProperties{
		Inputs:  f.Inputs,
		Outputs: f.Outputs,
	})
}

// String returns the string representation of the function.
func (f Function) String() string { return f.StringWithPrefix("") }

// StringWithPrefix returns the string representation with tree formatting.
func (f Function) StringWithPrefix(prefix string) string {
	var b strings.Builder
	b.WriteString(f.Key)
	b.WriteString("\n")

	hasInputs := len(f.Inputs) > 0
	hasOutputs := len(f.Outputs) > 0

	// Channels
	isLast := !hasInputs && !hasOutputs
	b.WriteString(prefix)
	b.WriteString(tree.Prefix(isLast))
	b.WriteString("channels: ")
	b.WriteString(f.Channels.String())
	b.WriteString("\n")

	if hasInputs {
		isLast = !hasOutputs
		b.WriteString(prefix)
		b.WriteString(tree.Prefix(isLast))
		b.WriteString("inputs: ")
		b.WriteString(f.Inputs.String())
		b.WriteString("\n")
	}

	if hasOutputs {
		b.WriteString(prefix)
		b.WriteString(tree.Prefix(true))
		b.WriteString("outputs: ")
		b.WriteString(f.Outputs.String())
		b.WriteString("\n")
	}

	return b.String()
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase Go
// field names and new lowercase msgpack tag names for backward compatibility.
func (f *Function) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Function
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(f)); err != nil {
		return err
	}
	if len(f.Key) == 0 {
		var legacy struct {
			Key      string
			Body     Body
			Inputs   types.Params
			Outputs  types.Params
			Channels types.Channels
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		f.Key = legacy.Key
		f.Body = legacy.Body
		f.Inputs = legacy.Inputs
		f.Outputs = legacy.Outputs
		f.Channels = legacy.Channels
	}
	return nil
}
