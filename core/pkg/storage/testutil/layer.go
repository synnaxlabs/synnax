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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
)

// Layer is an in-memory storage layer that runs until it closes.
type Layer struct {
	*storage.Layer
	stop func() error
}

// OpenLayer opens an in-memory storage layer and runs it until Close or until ctx ends.
// ctx must outlive the Layer. The caller closes it.
func OpenLayer(ctx context.Context) Layer {
	l := lo.Must(storage.OpenLayer(ctx, storage.LayerConfig{InMemory: new(true)}))
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- l.Run(ctx) }()
	return Layer{Layer: l, stop: sync.OnceValue(func() error {
		cancel()
		return <-done
	})}
}

// Close stops the run of the Layer, then closes it. It shadows storage.Layer.Close.
func (l Layer) Close() error { return errors.Join(l.stop(), l.Layer.Close()) }
