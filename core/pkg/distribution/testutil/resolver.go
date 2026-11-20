// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
)

// records holds the channels that a Node creates, so that distribution-layer tests can
// write to free and virtual channels without a service layer. It is safe for concurrent
// use.
type records struct {
	byKey map[channel.Key]channel.Channel
	mu    sync.RWMutex
}

func (r *records) add(channels []channel.Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ch := range channels {
		r.byKey[ch.Key()] = ch
	}
}

// Resolve implements writer.ChannelResolver.
func (r *records) Resolve(
	_ context.Context,
	keys channel.Keys,
) (map[channel.Key]writer.Facts, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	found := make(map[channel.Key]writer.Facts, len(keys))
	for _, k := range keys {
		if ch, ok := r.byKey[k]; ok {
			found[k] = writer.FactsOf(ch)
		}
	}
	return found, nil
}
