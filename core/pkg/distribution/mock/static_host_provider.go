// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
)

type staticHostProvider struct{ node node.Node }

var _ cluster.HostProvider = staticHostProvider{}

// NewStaticHostProvider returns a cluster.HostProvider that always reports the node
// identified by key as the host.
func NewStaticHostProvider(key node.Key) cluster.HostProvider {
	return staticHostProvider{node: node.Node{Key: key}}
}

// Host returns the static host node.
func (s staticHostProvider) Host() node.Node { return s.node }

// HostKey returns the key of the static host node.
func (s staticHostProvider) HostKey() node.Key { return s.node.Key }
