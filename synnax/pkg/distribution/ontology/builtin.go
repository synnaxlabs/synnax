// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
)

// BuiltIn is a resource type that is built into the ontology.
const BuiltIn Type = "builtin"

// Root is the root resource in the ontology. All other resources are reachable by
// traversing the ontology from the root.
var Root = ID{Type: BuiltIn, Key: "root"}

type builtinService struct{}

var _ Service = (*builtinService)(nil)

func (b *builtinService) Schema() *Schema {
	return &Schema{
		Type:   BuiltIn,
		Fields: map[string]schema.Field{},
	}
}

func (b *builtinService) RetrieveEntity(_ context.Context, key string) (Entity, error) {
	switch key {
	case "root":
		return Entity{Name: "root"}, nil
	default:
		panic("[ontology] - builtin entity not found")
	}
}
