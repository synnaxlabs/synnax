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
	"context"
	"iter"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
)

var (
	// RootID is the root resource in the ontology. All other resources are reachable by
	// traversing the ontology from the root.
	RootID       = resource.ID{Type: resource.TypeBuiltin, Key: "root"}
	rootResource = Resource{ID: RootID, Name: "root"}
)

type builtinService struct{ observe.Noop[iter.Seq[Change]] }

var _ Service = (*builtinService)(nil)

func (*builtinService) Type() resource.Type { return resource.TypeBuiltin }

// Search implements Service. No builtin resource is searchable.
func (*builtinService) Search(
	context.Context,
	gorp.Tx,
	search.Query[resource.ID],
) (search.Result[resource.ID], error) {
	return search.Result[resource.ID]{}, nil
}

// RetrieveResource implements Service.
func (*builtinService) RetrieveResource(
	_ context.Context, key string, _ gorp.Tx,
) (Resource, error) {
	switch key {
	case "root":
		return rootResource, nil
	default:
		return Resource{}, errors.Wrapf(
			query.ErrNotFound, "builtin resource %q not found", key,
		)
	}
}
