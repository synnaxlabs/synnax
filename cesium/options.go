// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"github.com/synnaxlabs/x/binary"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"go.uber.org/zap"
)

type Option func(*options)

type options struct {
	dirname string
	fs      xfs.FS
	logger  *zap.Logger
	metaECD binary.EncoderDecoder
}

func (o *options) logArgs() []interface{} {
	return []interface{}{
		"dirname", o.dirname,
	}
}

func newOptions(dirname string, opts ...Option) *options {
	o := &options{dirname: dirname}
	for _, opt := range opts {
		opt(o)
	}
	mergeDefaultOptions(o)
	return o
}

func mergeDefaultOptions(o *options) {
	o.logger = override.Nil(zap.NewNop(), o.logger)
	o.metaECD = override.Nil[binary.EncoderDecoder](&binary.JSONEncoderDecoder{}, o.metaECD)
	o.fs = override.Nil[xfs.FS](xfs.Default, o.fs)
}

func WithFS(fs xfs.FS) Option {
	return func(o *options) {
		o.fs = fs
	}
}

func MemBacked() Option {
	return func(o *options) {
		o.fs = xfs.NewMem()
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}
