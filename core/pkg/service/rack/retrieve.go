// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTX       gorp.Tx
	search       *search.Index
	gorp         gorp.Retrieve[Key, Rack]
	hostProvider cluster.HostProvider
	searchTerm   string
}

// WhereNames filters racks by their names.
func (r Retrieve) WhereNames(names []string, opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(_ gorp.Context, rack *Rack) (bool, error) {
		return lo.Contains(names, rack.Name), nil
	}, opts...)
	return r
}

// WhereName filters racks by their name.
func (r Retrieve) WhereName(name string, opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(_ gorp.Context, rack *Rack) (bool, error) {
		return name == rack.Name, nil
	}, opts...)
	return r
}

// WhereEmbedded filters for racks that are embedded within the Synnax server.
func (r Retrieve) WhereEmbedded(embedded bool, opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(_ gorp.Context, rack *Rack) (bool, error) {
		return rack.Embedded == embedded, nil
	}, opts...)
	return r
}

// WhereNodeIsHost filters for racks that are bound to the provided node and are
// a gateway.
func (r Retrieve) WhereNodeIsHost(nodeIsHost bool, opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(_ gorp.Context, rack *Rack) (bool, error) {
		isNodeHost := rack.Key.Node() == r.hostProvider.HostKey()
		return isNodeHost == nodeIsHost, nil
	}, opts...)
	return r
}

// WhereIntegration filters for racks that support the provided integration.
func (r Retrieve) WhereIntegration(
	integration string,
	opts ...gorp.FilterOption,
) Retrieve {
	r.gorp = r.gorp.Where(func(_ gorp.Context, rack *Rack) (bool, error) {
		return slices.Contains(rack.Integrations, integration), nil
	}, opts...)
	return r
}

// WhereNode filters for racks that are embedded within the provided node.
func (r Retrieve) WhereNode(node cluster.NodeKey, opts ...gorp.FilterOption) Retrieve {
	r.gorp = r.gorp.Where(func(_ gorp.Context, rack *Rack) (bool, error) {
		return rack.Key.Node() == node, nil
	}, opts...)
	return r
}
