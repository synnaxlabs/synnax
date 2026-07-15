// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"regexp"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/gorp"
)

// MatchVirtual returns a filter for channels that are virtual if virtual is true, or
// are not virtual if virtual is false. Calculated channels are excluded from the
// virtual bucket even though they are stored with Virtual=true.
func MatchVirtual(virtual bool) Filter {
	return Match(func(_ gorp.Context, _ Retrieve, ch *Channel) (bool, error) {
		isVirtual := ch.Virtual && !ch.IsCalculated()
		return isVirtual == virtual, nil
	})
}

// MatchCalculated returns a filter for channels that have a non-empty Expression field.
func MatchCalculated() Filter {
	return Match(func(_ gorp.Context, _ Retrieve, ch *Channel) (bool, error) {
		return ch.IsCalculated(), nil
	})
}

// MatchNames returns a filter for channels whose Name matches any of the provided
// patterns. Each pattern may be a literal channel name or a Go regular expression;
// unanchored patterns are wrapped in ^...$ before compilation.
//
// When every input is a literal channel name, MatchNames routes through the per-Service
// name index (r.indexes.name) for an O(1) candidate-key lookup instead of a full scan.
// If any input contains regex metacharacters, MatchNames compiles each pattern and
// falls back to a scan that tests every decoded channel.
func MatchNames(names ...string) Filter {
	if len(names) > 0 && allLiteralNames(names) {
		return func(r Retrieve) gorp.Filter[Key, Channel] {
			return r.indexes.name.Filter(names...)
		}
	}
	matchers := make([]func(string) bool, len(names))
	for i, name := range names {
		matchers[i] = formatNameMatcher(name)
	}
	return Match(func(_ gorp.Context, _ Retrieve, ch *Channel) (bool, error) {
		return lo.SomeBy(matchers, func(matcher func(string) bool) bool {
			return matcher(ch.Name)
		}), nil
	})
}

// allLiteralNames reports whether every input is accepted by validNamePattern, the
// character set enforced during channel creation and renaming. A channel's stored Name
// always matches it, so such inputs are literal exact-match targets routable through
// the in-memory name index; anything else contains regex metacharacters and must fall
// back to the regex matcher to preserve the historical contract.
func allLiteralNames(names []string) bool {
	for _, n := range names {
		if !validNamePattern.MatchString(n) {
			return false
		}
	}
	return true
}

func formatNameMatcher(name string) func(name string) bool {
	pattern := name
	if !strings.HasPrefix(pattern, "^") && !strings.HasSuffix(pattern, "$") {
		pattern = "^" + pattern + "$"
	}
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return func(s string) bool { return s == name }
	}
	return rx.MatchString
}
