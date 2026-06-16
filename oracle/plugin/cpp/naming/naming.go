// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package naming holds C++ name derivations shared between the cpp/types and
// cpp/json plugins, so the header and its JSON codec cannot disagree on a name.
package naming

import "github.com/synnaxlabs/oracle/plugin/internal/casing"

// VariantTypeName returns the C++ struct name for one variant of a discriminated
// union (e.g. union "Scale" + value "linear" -> "ScaleLinear", union "AIChannel" +
// value "ai_voltage" -> "AIVoltageChannel"). cpp/types declares the variant
// struct under this name and cpp/json emits its parse/to_json bodies, so both
// MUST derive it identically; routing through this one function guarantees they
// cannot drift.
func VariantTypeName(unionCppName, variantValue string) string {
	return casing.VariantTypeName(unionCppName, variantValue)
}
