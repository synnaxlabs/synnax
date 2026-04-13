// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package filter provides shared comparison structs for the typed query protocol.
// These structs are pure data containers for deserializing filter trees from the
// wire format (JSON/msgpack). Interpretation into gorp.Filter values is handled
// by oracle-generated code in each entity's API package.
package filter

import "github.com/synnaxlabs/x/telem"

// StringFilter holds comparison operators for a string-valued field.
type StringFilter struct {
	Eq       *string  `json:"eq,omitempty" msgpack:"eq,omitempty"`
	In       []string `json:"in,omitempty" msgpack:"in,omitempty"`
	Contains *string  `json:"contains,omitempty" msgpack:"contains,omitempty"`
	Regex    *string  `json:"regex,omitempty" msgpack:"regex,omitempty"`
}

// BoolFilter holds comparison operators for a bool-valued field.
type BoolFilter struct {
	Eq *bool `json:"eq,omitempty" msgpack:"eq,omitempty"`
}

// NumericFilter holds comparison operators for an ordered numeric field.
type NumericFilter[T ~int8 | ~int16 | ~int32 | ~int64 |
	~uint8 | ~uint16 | ~uint32 | ~uint64 |
	~float32 | ~float64] struct {
	Eq  *T  `json:"eq,omitempty" msgpack:"eq,omitempty"`
	In  []T `json:"in,omitempty" msgpack:"in,omitempty"`
	Gt  *T  `json:"gt,omitempty" msgpack:"gt,omitempty"`
	Lt  *T  `json:"lt,omitempty" msgpack:"lt,omitempty"`
	Gte *T  `json:"gte,omitempty" msgpack:"gte,omitempty"`
	Lte *T  `json:"lte,omitempty" msgpack:"lte,omitempty"`
}

// TimeFilter holds comparison operators for a telem.TimeStamp-valued field.
type TimeFilter struct {
	Eq  *telem.TimeStamp `json:"eq,omitempty" msgpack:"eq,omitempty"`
	Gt  *telem.TimeStamp `json:"gt,omitempty" msgpack:"gt,omitempty"`
	Lt  *telem.TimeStamp `json:"lt,omitempty" msgpack:"lt,omitempty"`
	Gte *telem.TimeStamp `json:"gte,omitempty" msgpack:"gte,omitempty"`
	Lte *telem.TimeStamp `json:"lte,omitempty" msgpack:"lte,omitempty"`
}
