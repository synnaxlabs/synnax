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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/x/gorp"
)

type Retrieve struct {
	baseTX       gorp.Tx
	search       *search.Index
	gorp         gorp.Retrieve[Key, Rack]
	hostProvider node.HostProvider
	searchTerm   string
}

// MatchNodeIsHost returns a filter that matches racks whose node is (or is
// not) the current host, using the host provider held on the Retrieve.
func MatchNodeIsHost(v bool) Filter {
	return Match(func(_ gorp.Context, r Retrieve, rack *Rack) (bool, error) {
		isNodeHost := rack.Key.Node() == r.hostProvider.HostKey()
		return isNodeHost == v, nil
	})
}

// MatchNode returns a filter that matches racks on the given cluster node.
func MatchNode(node node.Key) Filter {
	return func(_ Retrieve) gorp.Filter[Key, Rack] {
		return gorp.Match(func(_ gorp.Context, rack *Rack) (bool, error) {
			return rack.Key.Node() == node, nil
		})
	}
}
