// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package mock provides service-layer channel test helpers for use with a
// distribution-layer mock cluster. It lives in the service layer because it opens a
// real service-layer channel.Service (which the distribution layer must not depend on).
package mock

import (
	"context"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/distribution"
	distmock "github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
)

var (
	mu       sync.Mutex
	services = make(map[*distribution.Layer]*channel.Service)
)

// ChannelService returns the service-layer channel.Service for the given mock node,
// opening it (via channel.Wrap, which creates the node's control channel) on first use
// and caching it per node thereafter. Any WrapOptions are applied only on the first call
// for a given node; later calls return the cached service and ignore the options.
func ChannelService(n distmock.Node, opts ...channel.WrapOption) *channel.Service {
	mu.Lock()
	defer mu.Unlock()
	if s, ok := services[n.Layer]; ok {
		return s
	}
	s := channel.Wrap(n.Layer, opts...)
	services[n.Layer] = s
	return s
}

// OpenWriter resolves the distribution-layer channel metadata for cfg.Keys via svc (the
// test's service-layer channel service) and opens a distribution writer against n. Since
// the distribution writer no longer resolves channels itself, service-layer tests use
// this to supply the resolved channels their service-created keys require.
func OpenWriter(
	ctx context.Context,
	n distmock.Node,
	svc *channel.Service,
	cfg writer.Config,
) (*writer.Writer, error) {
	channels, err := svc.RetrieveByKeys(ctx, cfg.Keys...)
	if err != nil {
		return nil, err
	}
	cfg.Channels = channels
	return n.Framer.OpenWriter(ctx, cfg)
}
