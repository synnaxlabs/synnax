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
func MatchVirtual(virtual bool) gorp.Filter[Key, Channel] {
	return gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
		isVirtual := ch.Virtual && !ch.IsCalculated()
		return isVirtual == virtual, nil
	})
}

// MatchCalculated returns a filter for channels that have a non-empty Expression
// field.
func MatchCalculated() gorp.Filter[Key, Channel] {
	return gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
		return ch.IsCalculated(), nil
	})
}

// MatchNames returns a filter for channels whose Name attribute matches any of the
// provided name patterns. Each pattern may be a literal name or a regular expression;
// a pattern that is neither anchored with ^ nor $ is wrapped in ^...$ before
// compilation.
func MatchNames(names ...string) gorp.Filter[Key, Channel] {
	matchers := make([]func(string) bool, len(names))
	for i, name := range names {
		matchers[i] = formatNameMatcher(name)
	}
	return gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
		return lo.SomeBy(matchers, func(matcher func(string) bool) bool {
			return matcher(ch.Name)
		}), nil
	})
}

func formatNameMatcher(name string) func(name string) bool {
	if !strings.HasPrefix(name, "^") && !strings.HasSuffix(name, "$") {
		name = "^" + name + "$"
	}
	rx, err := regexp.Compile(name)
	if err != nil {
		return func(s string) bool { return s == name }
	}
	return rx.MatchString
}
