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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/binary"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
)

type Option func(*options)

type options struct {
	alamos.Instrumentation
	dirname string
	fs      xfs.FS
	metaECD binary.EncoderDecoder
	gcCfg   *GCConfig
}

func (o *options) Report() alamos.Report {
	return alamos.Report{
		"dirname": o.dirname,
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
	o.metaECD = override.Nil[binary.EncoderDecoder](&binary.JSONEncoderDecoder{}, o.metaECD)
	o.fs = override.Nil[xfs.FS](xfs.Default, o.fs)
	o.gcCfg = override.Nil[*GCConfig](&DefaultGCConfig, o.gcCfg)
}

func WithFS(fs xfs.FS) Option {
	return func(o *options) {
		o.fs = fs
	}
}

func WithGC(config *GCConfig) Option {
	return func(o *options) {
		o.gcCfg = config
	}
}

func MemBacked() Option {
	return func(o *options) {
		o.fs = xfs.NewMem()
	}
}

func WithInstrumentation(i alamos.Instrumentation) Option {
	return func(o *options) {
		o.Instrumentation = i
	}
}
