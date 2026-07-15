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
	v2 "github.com/synnaxlabs/arc/ir/types/v2"
	"github.com/synnaxlabs/arc/types"
)

// TreePrefix returns the prefix for a tree item. If last is true, returns "└── ",
// otherwise "├── ".
func TreePrefix(last bool) string { return v2.TreePrefix(last) }

// FormatFunctionSignature returns a human-readable function signature in Arc syntax.
// Format: "name(param type, param type) returnType"
func FormatFunctionSignature(name string, t types.Type) string {
	return v2.FormatFunctionSignature(name, t)
}
