// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import "fmt"

// Key returns the member's lookup key — the string transitions target via `=> name`.
// Derived from the set variant: the referenced node's key for leaf members, the nested
// scope's key for scope members. Returns the empty string for an unset member.
func (m Member) Key() string {
	switch {
	case m.NodeKey != nil:
		return *m.NodeKey
	case m.Scope != nil:
		return m.Scope.Key
	default:
		return ""
	}
}

// String returns the tree representation of a Member.
func (m Member) String() string { return m.stringWithPrefix("") }

func (m Member) stringWithPrefix(prefix string) string {
	switch {
	case m.NodeKey != nil:
		return fmt.Sprintf("%s\n", *m.NodeKey)
	case m.Scope != nil:
		return m.Scope.StringWithPrefix(prefix)
	default:
		return "(empty member)\n"
	}
}
