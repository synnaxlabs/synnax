// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"fmt"
	"strings"

	"github.com/synnaxlabs/arc/types"
	"github.com/vmihailenco/msgpack/v5"
)

// IsEntryNode reports whether n is an entry node: it has no incoming edges and
// reads no channels. Entry nodes fire once per activation.
func (n Node) IsEntryNode(edges Edges) bool {
	return len(edges.GetInputs(n.Key)) == 0 && len(n.Channels.Read) == 0
}

// String returns the string representation of the node.
func (n Node) String() string { return n.stringWithPrefix("") }

// stringWithPrefix returns the string representation with tree formatting.
func (n Node) stringWithPrefix(prefix string) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%s (type: %s)\n", n.Key, n.Type)

	hasInputs := len(n.Inputs) > 0
	hasOutputs := len(n.Outputs) > 0

	isLast := !hasInputs && !hasOutputs
	b.WriteString(prefix)
	b.WriteString(treePrefix(isLast))
	b.WriteString("channels: ")
	b.WriteString(formatChannels(n.Channels))
	b.WriteString("\n")

	if hasInputs {
		isLast = !hasOutputs
		b.WriteString(prefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString("inputs: ")
		b.WriteString(formatParams(n.Inputs))
		b.WriteString("\n")
	}

	if hasOutputs {
		b.WriteString(prefix)
		b.WriteString(treePrefix(true))
		b.WriteString("outputs: ")
		b.WriteString(formatParams(n.Outputs))
		b.WriteString("\n")
	}

	return b.String()
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase Go
// field names and new lowercase msgpack tag names for backward compatibility.
func (n *Node) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Node
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(n)); err != nil {
		return err
	}
	if len(n.Key) == 0 {
		var legacy struct {
			Key      string
			Type     string
			Inputs   types.Params
			Outputs  types.Params
			Channels types.Channels
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		n.Key = legacy.Key
		n.Type = legacy.Type
		n.Inputs = legacy.Inputs
		n.Outputs = legacy.Outputs
		n.Channels = legacy.Channels
	}
	return nil
}
