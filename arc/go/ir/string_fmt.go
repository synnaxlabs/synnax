// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import (
	"slices"
	"strings"

	"github.com/synnaxlabs/arc/fmtstring"
	"github.com/synnaxlabs/arc/types"
)

// stringFmtNodeType matches the symbol resolver entry in
// arc/go/stl/strings/string.go.
const stringFmtNodeType = "string.fmt"

// StringFmtSyntheticPrefix tags Function entries synthesized for flow-form
// string.fmt nodes with placeholders, gating synthetic emission in the
// compiler.
const StringFmtSyntheticPrefix = "fmt$"

// RewriteStringFmtNodes appends a synthetic Function for each flow-form
// string.fmt node whose format literal contains placeholders, rewrites the
// node's Type to the synthetic key, and clears Inputs/Config. Placeholder-free
// nodes are left untouched for the fmtNode runtime path.
func RewriteStringFmtNodes(nodes Nodes, functions *Functions) {
	for idx := range nodes {
		n := &nodes[idx]
		if n.Type != stringFmtNodeType {
			continue
		}
		formatParam, ok := n.Config.Get("format")
		if !ok {
			continue
		}
		format, ok := formatParam.Value.(string)
		if !ok {
			continue
		}
		segments, err := fmtstring.Parse(format)
		if err != nil {
			continue
		}
		hasPlaceholder := false
		for _, seg := range segments {
			if seg.IsPlaceholder {
				hasPlaceholder = true
				break
			}
		}
		if !hasPlaceholder {
			continue
		}
		// Sanitize "." in n.Key: CompoundFactory.Create splits on the last
		// "." as a module prefix, which would mis-route the synthetic type.
		//
		// TODO: once backtick literals implicitly desugar to a non-public,
		// single-segment fmt call (no module qualification), n.Key will not
		// contain "." and the sanitization can be removed. Revert to:
		//     synthKey := StringFmtSyntheticPrefix + n.Key
		synthKey := StringFmtSyntheticPrefix + strings.ReplaceAll(n.Key, ".", "_")
		*functions = append(*functions, Function{
			Key:      synthKey,
			Body:     Body{Raw: format},
			Inputs:   types.Params{},
			Config:   types.Params{},
			Outputs:  slices.Clone(n.Outputs),
			Channels: types.NewChannels(),
		})
		n.Type = synthKey
		n.Inputs = types.Params{}
		n.Config = types.Params{}
	}
}

// IsStringFmtSyntheticKey reports whether key names a synthetic Function
func IsStringFmtSyntheticKey(key string) bool {
	return strings.HasPrefix(key, StringFmtSyntheticPrefix)
}
