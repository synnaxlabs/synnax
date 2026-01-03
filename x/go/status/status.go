// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"fmt"
	"strings"

	"github.com/synnaxlabs/x/gorp"
)

// String returns a formatted string representation of the Status.
func (s Status[D]) String() string {
	var b strings.Builder

	var variantIcon string
	switch s.Variant {
	case VariantInfo:
		variantIcon = "ℹ"
	case VariantSuccess:
		variantIcon = "✓"
	case VariantError:
		variantIcon = "✗"
	case VariantWarning:
		variantIcon = "⚠"
	case VariantDisabled:
		variantIcon = "⊘"
	case VariantLoading:
		variantIcon = "◌"
	default:
		variantIcon = "•"
	}

	_, _ = fmt.Fprintf(&b, "[%s %s]", variantIcon, s.Variant)

	if s.Name != "" {
		_, _ = fmt.Fprintf(&b, " %s", s.Name)
	}

	if s.Key != "" && s.Key != s.Name {
		_, _ = fmt.Fprintf(&b, " (%s)", s.Key)
	}

	if s.Message != "" {
		_, _ = fmt.Fprintf(&b, ": %s", s.Message)
	}

	if s.Description != "" {
		_, _ = fmt.Fprintf(&b, "\n  %s", s.Description)
	}

	if s.Time != 0 {
		_, _ = fmt.Fprintf(&b, "\n  @ %s", s.Time)
	}

	if detailStr := fmt.Sprintf("%v", s.Details); detailStr != "" && detailStr != "<nil>" && detailStr != "0" {
		var zero D
		if fmt.Sprintf("%v", zero) != detailStr {
			_, _ = fmt.Fprintf(&b, "\n  Details: %v", s.Details)
		}
	}

	return b.String()
}

var _ gorp.Entry[string] = (*Status[any])(nil)

// GorpKey implements gorp.Entry.
func (s Status[D]) GorpKey() string { return s.Key }

// SetOptions implements gorp.Entry.
func (s Status[D]) SetOptions() []any { return nil }

// CustomTypeName implements types.CustomTypeName to ensure that Status struct does
// not conflict with any other types in gorp.
func (s Status[D]) CustomTypeName() string { return "Status" }
