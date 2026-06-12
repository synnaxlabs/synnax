// Copyright 2026 Synnax Labs, Inc.
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
	xstatus "github.com/synnaxlabs/x/status"
)

var _ gorp.Entry[string] = (*Status[any])(nil)

// GorpKey implements gorp.Entry.
func (s Status[D]) GorpKey() string { return s.Key }

// SetOptions implements gorp.Entry.
func (s Status[D]) SetOptions() []any { return nil }

// CustomTypeName implements types.CustomTypeName to ensure that Status struct does
// not conflict with any other types in gorp.
func (s Status[D]) CustomTypeName() string { return "Status" }

// String returns a formatted string representation of the Status.
func (s Status[D]) String() string {
	var b strings.Builder

	var variantIcon string
	switch s.Variant {
	case xstatus.VariantInfo:
		variantIcon = "ℹ"
	case xstatus.VariantSuccess:
		variantIcon = "✓"
	case xstatus.VariantError:
		variantIcon = "✗"
	case xstatus.VariantWarning:
		variantIcon = "⚠"
	case xstatus.VariantDisabled:
		variantIcon = "⊘"
	case xstatus.VariantLoading:
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
