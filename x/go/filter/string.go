// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package filter

import (
	"slices"
	"strings"
)

// MatchesString reports whether v matches every operator of s but Regex, which the
// caller compiles. With CaseInsensitive set, it compares lower-case values.
func MatchesString(s String, v string) bool {
	fold := func(x string) string {
		if s.CaseInsensitive {
			return strings.ToLower(x)
		}
		return x
	}
	v = fold(v)
	return (s.Eq == nil || fold(*s.Eq) == v) &&
		(s.In == nil || slices.ContainsFunc(s.In, func(e string) bool {
			return fold(e) == v
		})) &&
		(s.Contains == nil || strings.Contains(v, fold(*s.Contains))) &&
		(s.StartsWith == nil || strings.HasPrefix(v, fold(*s.StartsWith)))
}
