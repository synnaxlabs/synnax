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

func WhereName(name string) gorp.Filter[Key, Rack] {
	return gorp.Match[Key, Rack](func(_ gorp.Context, rack *Rack) (bool, error) {
		return name == rack.Name, nil
	})
}

func WhereNodeIsHost(
	host cluster.HostProvider,
	v bool,
) gorp.Filter[Key, Rack] {
	return gorp.Match[Key, Rack](func(_ gorp.Context, rack *Rack) (bool, error) {
		isNodeHost := rack.Key.Node() == host.HostKey()
		return isNodeHost == v, nil
	})
}

func WhereIntegration(integration string) gorp.Filter[Key, Rack] {
	return gorp.Match[Key, Rack](func(_ gorp.Context, rack *Rack) (bool, error) {
		return slices.Contains(rack.Integrations, integration), nil
	})
}

func WhereNode(node cluster.NodeKey) gorp.Filter[Key, Rack] {
	return gorp.Match[Key, Rack](func(_ gorp.Context, rack *Rack) (bool, error) {
		return rack.Key.Node() == node, nil
	})
}
