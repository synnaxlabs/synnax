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
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Retrieve is used to retrieve information about Channel(s) in the synnax distribution
// layer. It is a thin wrapper around gorp.Retrieve that adds channel-specific filter
// functions, fuzzy search, and cluster-level validation. The generated retrieve.gen.go
// file provides Filter, Match/And/Or/Not, Where, WhereKeys, Entry, Entries, Limit,
// Offset, Search, Exec, Count, and Exists.
type Retrieve struct {
	baseTX     gorp.Tx
	gorp       gorp.Retrieve[Key, Channel]
	search     *search.Index
	searchTerm string
	indexes    indexes
}

// MatchNodeKey returns a filter for channels whose Leaseholder attribute matches the
// provided leaseholder node Key.
func MatchNodeKey(nodeKey cluster.NodeKey) Filter {
	return func(_ Retrieve) gorp.Filter[Key, Channel] {
		return gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
			return ch.Leaseholder == nodeKey, nil
		})
	}
}

// MatchIsIndex returns a filter for channels that are indexes if isIndex is true, or
// are not indexes if isIndex is false.
func MatchIsIndex(isIndex bool) Filter {
	return func(_ Retrieve) gorp.Filter[Key, Channel] {
		return gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
			return ch.IsIndex == isIndex, nil
		})
	}
}

// MatchVirtual returns a filter for channels that are virtual if virtual is true, or
// are not virtual if virtual is false. Calculated channels are excluded from the
// virtual bucket even though they are stored with Virtual=true.
func MatchVirtual(virtual bool) Filter {
	return func(_ Retrieve) gorp.Filter[Key, Channel] {
		return gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
			isVirtual := ch.Virtual && !ch.IsCalculated()
			return isVirtual == virtual, nil
		})
	}
}

// MatchInternal returns a filter for channels that are internal if internal is true,
// or are not internal if internal is false.
func MatchInternal(internal bool) Filter {
	return func(_ Retrieve) gorp.Filter[Key, Channel] {
		return gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
			return ch.Internal == internal, nil
		})
	}
}

// MatchDataTypes returns a filter for channels whose DataType attribute matches any
// of the provided data types.
func MatchDataTypes(dataTypes ...telem.DataType) Filter {
	return func(_ Retrieve) gorp.Filter[Key, Channel] {
		return gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
			return lo.Contains(dataTypes, ch.DataType), nil
		})
	}
}

// MatchNotDataTypes returns a filter for channels whose DataType attribute does not
// match any of the provided data types.
func MatchNotDataTypes(dataTypes ...telem.DataType) Filter {
	return func(_ Retrieve) gorp.Filter[Key, Channel] {
		return gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
			return !lo.Contains(dataTypes, ch.DataType), nil
		})
	}
}

// MatchCalculated returns a filter for channels that have a non-empty Expression
// field.
func MatchCalculated() Filter {
	return func(_ Retrieve) gorp.Filter[Key, Channel] {
		return gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
			return ch.IsCalculated(), nil
		})
	}
}

// literalNamePattern matches the character set enforced by ValidateName. A
// channel's stored Name is always accepted by this regex, so any input that
// passes this check is a literal exact-match target and can be routed through
// the in-memory name index instead of a scan. Any input that fails the check
// contains regex metacharacters (., *, ?, brackets, anchors) and must fall
// back to the regex matcher to preserve the historical contract.
var literalNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// MatchNames returns a filter for channels whose Name matches any of the
// provided patterns. Each pattern may be a literal channel name or a Go
// regular expression; unanchored patterns are wrapped in ^...$ before
// compilation.
//
// When every input is a literal channel name, MatchNames routes through the
// per-Service name index (r.indexes.name) for an O(1) candidate-key lookup
// instead of a full scan. If any input contains regex metacharacters,
// MatchNames compiles each pattern and falls back to a scan that tests every
// decoded channel.
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
	return func(_ Retrieve) gorp.Filter[Key, Channel] {
		return gorp.Match(func(_ gorp.Context, ch *Channel) (bool, error) {
			return lo.SomeBy(matchers, func(matcher func(string) bool) bool {
				return matcher(ch.Name)
			}), nil
		})
	}
}

func allLiteralNames(names []string) bool {
	for _, n := range names {
		if !literalNamePattern.MatchString(n) {
			return false
		}
	}
	return true
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
