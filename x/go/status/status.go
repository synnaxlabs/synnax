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

	"github.com/synnaxlabs/x/telem"
)

// Variant is a general classification mechanism for statuses.
type Variant string

const (
	InfoVariant     Variant = "info"
	SuccessVariant  Variant = "success"
	ErrorVariant    Variant = "error"
	WarningVariant  Variant = "warning"
	DisabledVariant Variant = "disabled"
	LoadingVariant  Variant = "loading"
)

// Status is a standardized payload used across Synnax.
type Status[D any] struct {
	// Key is a unique key for the status.
	Key string `json:"key" msgpack:"key"`
	// Name is a human-readable name for the status.
	Name string `json:"name" msgpack:"name"`
	// Variant is the variant of the status.
	Variant Variant `json:"variant" msgpack:"variant"`
	// Message is the message of the status.
	Message string `json:"message" msgpack:"message"`
	// Description is the description of the status.
	Description string `json:"description" msgpack:"description"`
	// Time is the time the status was created.
	Time telem.TimeStamp `json:"time" msgpack:"time"`
	// Details are customizable details for component specific statuses.
	Details D `json:"details" msgpack:"details"`
}

// String returns a formatted string representation of the Status.
func (s Status[D]) String() string {
	var b strings.Builder

	var variantIcon string
	switch s.Variant {
	case InfoVariant:
		variantIcon = "ℹ"
	case SuccessVariant:
		variantIcon = "✓"
	case ErrorVariant:
		variantIcon = "✗"
	case WarningVariant:
		variantIcon = "⚠"
	case DisabledVariant:
		variantIcon = "⊘"
	case LoadingVariant:
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
