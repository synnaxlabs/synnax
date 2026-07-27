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
	"strings"

	"github.com/synnaxlabs/arc/types"
)

// FormatFunctionSignature returns a human-readable function signature in Arc syntax.
// Format: "name(param type, param type) returnType"
func FormatFunctionSignature(name string, t types.Type) string {
	if t.Kind != types.KindFunction {
		return name
	}
	var sb strings.Builder
	sb.WriteString(name)
	sb.WriteString("(")
	for i, p := range t.Inputs {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p.Name)
		sb.WriteString(" ")
		sb.WriteString(p.Type.String())
	}
	sb.WriteString(")")
	if len(t.Outputs) > 0 {
		sb.WriteString(" ")
		if len(t.Outputs) == 1 && t.Outputs[0].Name == DefaultOutputParam {
			sb.WriteString(t.Outputs[0].Type.String())
		} else if len(t.Outputs) == 1 {
			sb.WriteString(t.Outputs[0].Name)
			sb.WriteString(" ")
			sb.WriteString(t.Outputs[0].Type.String())
		} else {
			sb.WriteString("(")
			for i, p := range t.Outputs {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(p.Name)
				sb.WriteString(" ")
				sb.WriteString(p.Type.String())
			}
			sb.WriteString(")")
		}
	}
	return sb.String()
}
