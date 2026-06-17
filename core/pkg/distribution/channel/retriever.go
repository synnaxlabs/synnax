// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"sync"
)

// Retriever is the narrow read interface the distribution layer (chiefly the framer)
// needs against channel metadata. The rich channel record and its table live in the
// service layer, which implements this interface and binds it into distribution
// consumers after open. Retriever returns minimal, distribution-layer channels.
type Retriever interface {
	// RetrieveByKeys returns the channels matching the provided keys. Keys that do not
	// resolve to a channel are omitted from the result.
	RetrieveByKeys(context.Context, ...Key) ([]Channel, error)
}

// RetrieverHolder is a settable, concurrency-safe Retriever indirection. The
// distribution layer constructs a holder and hands it to consumers (e.g. the framer) at
// open time; the service layer binds the real Retriever into it once its channel
// service is available. Reads occur lazily per request, so binding before the server
// serves traffic is sufficient.
type RetrieverHolder struct {
	mu sync.RWMutex
	r  Retriever
}

// NewRetrieverHolder returns an empty RetrieverHolder.
func NewRetrieverHolder() *RetrieverHolder { return &RetrieverHolder{} }

// Bind sets the backing Retriever.
func (h *RetrieverHolder) Bind(r Retriever) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.r = r
}

func (h *RetrieverHolder) retriever() Retriever {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.r
}

// RetrieveByKeys implements Retriever.
func (h *RetrieverHolder) RetrieveByKeys(ctx context.Context, keys ...Key) ([]Channel, error) {
	return h.retriever().RetrieveByKeys(ctx, keys...)
}

var _ Retriever = (*RetrieverHolder)(nil)

// TryToRetrieveStringer returns a human-readable representation of the channel with the
// given key, falling back to the key's string form if it cannot be retrieved.
func TryToRetrieveStringer(ctx context.Context, r Retriever, key Key) string {
	if r == nil {
		return key.String()
	}
	chs, err := r.RetrieveByKeys(ctx, key)
	if err != nil || len(chs) == 0 {
		return key.String()
	}
	return chs[0].String()
}
