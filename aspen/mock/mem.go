// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"time"

	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/aspen/transport/mock"
)

func NewMemBuilder(defaultOpts ...aspen.Option) *Builder {
	propConfig := aspen.PropagationConfig{
		PledgeRetryInterval:   10 * time.Millisecond,
		PledgeRetryScale:      1,
		ClusterGossipInterval: 50 * time.Millisecond,
		KVGossipInterval:      50 * time.Millisecond,
	}
	return &Builder{
		DefaultOptions: append([]aspen.Option{
			aspen.WithTransport(mock.NewNetwork().NewTransport()),
			aspen.MemBacked(),
			aspen.WithPropagationConfig(propConfig),
		}, defaultOpts...),
		Nodes:     make(map[aspen.NodeKey]NodeInfo),
		memBacked: true,
	}
}
