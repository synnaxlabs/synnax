// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/zyn"
)

// NewResource creates a new Resource with the given schema, name, and data. NewResource
// panics if the provided data value does not fit the Resource's schema.
func NewResource(schema zyn.Schema, id resource.ID, name string, data any) Resource {
	return Resource{
		ID:   id,
		Name: name,
		Data: lo.Must(schema.Dump(data)),
	}
}

// ResourceIDs extracts the IDs from a slice of Resources.
func ResourceIDs(resources []Resource) []resource.ID {
	return lo.Map(resources, func(r Resource, _ int) resource.ID { return r.ID })
}

// Change is a change to a Resource.
type Change = change.Change[string, Resource]
