// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signal

import (
	"github.com/synnaxlabs/alamos"
	"go.uber.org/zap"
)

type Option func(o *options)

type options struct {
	alamos.Instrumentation
}

func WithInstrumentation(i alamos.Instrumentation) Option {
	return func(o *options) {
		i.L = i.L.WithOptions(zap.AddCallerSkip(0))
		o.Instrumentation = i
	}
}

func newOptions(opts []Option) options {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return *o
}
