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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/node"
)

type Retrieve struct {
	indexes indexes
	gorp    gorp.Retrieve[Key, Rack]
}

// MatchNode returns a filter that matches racks on the given node.
func MatchNode(node node.Key) Filter {
	return func(_ Retrieve) gorp.Filter[Key, Rack] {
		return gorp.Match(func(_ gorp.Context, rack *Rack) (bool, error) {
			return rack.Key.Node() == node, nil
		})
	}
}
