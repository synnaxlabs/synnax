// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

// IsZero reports whether the scope carries no execution content.
func (s Scope) IsZero() bool {
	return s.Key == "" &&
		s.Mode == ScopeModeUnspecified &&
		s.Liveness == LivenessUnspecified &&
		s.Activation == nil &&
		len(s.Strata) == 0 &&
		len(s.Steps) == 0 &&
		len(s.Transitions) == 0
}

// String returns the tree representation of a Scope.
func (s Scope) String() string { return s.StringWithPrefix("") }

// StringWithPrefix returns the tree representation with each line indented by prefix.
// Exported so parent printers in successor versions can nest it.
func (s Scope) StringWithPrefix(prefix string) string {
	var b strings.Builder
	lo.Must(fmt.Fprintf(&b, "%s [%s, %s]\n", scopeLabel(s), s.Mode, s.Liveness))
	if s.Mode == ScopeModeParallel {
		for i, stratum := range s.Strata {
			isLast := i == len(s.Strata)-1 && len(s.Transitions) == 0
			b.WriteString(prefix)
			b.WriteString(treePrefix(isLast))
			lo.Must(fmt.Fprintf(&b, "stratum %d\n", i))
			childPrefix := prefix + treeIndent(isLast)
			for j, m := range stratum {
				isLastMember := j == len(stratum)-1
				b.WriteString(childPrefix)
				b.WriteString(treePrefix(isLastMember))
				b.WriteString(m.stringWithPrefix(childPrefix + treeIndent(isLastMember)))
			}
		}
	} else {
		for i, m := range s.Steps {
			isLast := i == len(s.Steps)-1 && len(s.Transitions) == 0
			b.WriteString(prefix)
			b.WriteString(treePrefix(isLast))
			b.WriteString(m.stringWithPrefix(prefix + treeIndent(isLast)))
		}
	}
	for i, t := range s.Transitions {
		isLast := i == len(s.Transitions)-1
		b.WriteString(prefix)
		b.WriteString(treePrefix(isLast))
		b.WriteString(t.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func scopeLabel(s Scope) string {
	if s.Key == "" {
		return "(scope)"
	}
	return s.Key
}

// treePrefix returns the prefix for a tree item. If last is true, returns "└── ",
// otherwise "├── ".
func treePrefix(last bool) string {
	if last {
		return "└── "
	}
	return "├── "
}

func treeIndent(last bool) string {
	if last {
		return "    "
	}
	return "│   "
}
