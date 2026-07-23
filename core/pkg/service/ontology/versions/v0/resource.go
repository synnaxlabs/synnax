// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/synnaxlabs/x/gorp"

// Resource represents an instance matching of a resource in the ontology.
type Resource struct {
	// Data is the data for the Resource. Data must be parseable by the Resource's
	// schema.
	Data any `json:"data" msgpack:"data"`
	// ID is the unique identifier for the Resource.
	ID ID `json:"id" msgpack:"id"`
	// Name is a human-readable name for the Resource.
	Name string `json:"name" msgpack:"name"`
}

var _ gorp.Entry[string] = Resource{}

// GorpKey implements gorp.Entry.
func (r Resource) GorpKey() string { return r.ID.String() }

// SetOptions implements gorp.Entry.
func (Resource) SetOptions() []any { return nil }
