// Copyright 2025 Synnax Labs, Inc.
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
	"io"
	"iter"
	"slices"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/zyn"
)

// TypeBuiltIn is a resource type that is built into the ontology.
const TypeBuiltIn Type = "builtin"

var (
	// RootID is the root resource in the ontology. All other resources are reachable by
	// traversing the ontology from the root.
	RootID       = ID{Type: TypeBuiltIn, Key: "root"}
	rootResource = Resource{ID: RootID, Name: "root"}
)

type builtinService struct{ observe.Noop[iter.Seq[Change]] }

var _ Service = (*builtinService)(nil)

func (b *builtinService) Type() Type { return TypeBuiltIn }

// Schema implements Service.
func (b *builtinService) Schema() zyn.Schema { return zyn.Object(nil) }

// RetrieveResource implements Service.
func (b *builtinService) RetrieveResource(_ context.Context, key string, _ gorp.Tx) (Resource, error) {
	switch key {
	case "root":
		return rootResource, nil
	default:
		return Resource{}, errors.Wrapf(query.NotFound, "builtin resource %q not found", key)
	}
}

// OpenNexter implements Service.
func (b *builtinService) OpenNexter(context.Context) (iter.Seq[Resource], io.Closer, error) {
	return slices.Values([]Resource{rootResource}), xio.NopCloser, nil
}
