// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import "strings"

// StringFmtSyntheticPrefix tags Function entries synthesized for backtick
// literals with placeholders, gating synthetic emission in the compiler.
const StringFmtSyntheticPrefix = "fmt$"

// IsStringFmtSyntheticKey reports whether key names a synthetic Function.
func IsStringFmtSyntheticKey(key string) bool {
	return strings.HasPrefix(key, StringFmtSyntheticPrefix)
}
