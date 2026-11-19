// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	jsonv2 "encoding/json/v2"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	xtime "github.com/synnaxlabs/x/time"
	"github.com/synnaxlabs/x/validate"
)

type Option func(*options)

// defaultChunkSize keeps a chunk of a fixed-span read at about 64 MB of 8-byte samples.
const defaultChunkSize = 1 << 23

type options struct {
	alamos.Instrumentation
	fs        xfs.FS
	metaCodec encoding.Codec
	dirname   string
	gcCfg     GCConfig
	fileSize  telem.Size
	chunkSize int64
	clock     xtime.Clock
}

func (o *options) Report() alamos.Report { return alamos.Report{"dirname": o.dirname} }

func newOptions(dirname string, opts ...Option) (*options, error) {
	o := &options{
		dirname:   dirname,
		chunkSize: defaultChunkSize,
	}
	for _, opt := range opts {
		opt(o)
	}
	if err := mergeAndValidateOptions(o); err != nil {
		return nil, err
	}
	return o, nil
}

func mergeAndValidateOptions(o *options) error {
	// The default codec matches object names case-insensitively because databases
	// written before Channel.Virtual and Channel.Concurrency carried json tags stored
	// them under their Go field names.
	o.metaCodec = override.Nil[encoding.Codec](
		json.NewCodec(jsonv2.MatchCaseInsensitiveNames(true)),
		o.metaCodec,
	)
	o.fs = override.Nil(xfs.Default, o.fs)
	gcCfg := GCConfig{
		MaxGoroutine: 10,
		TryInterval:  30 * time.Second,
		Threshold:    0.2,
	}.Override(o.gcCfg)
	if err := gcCfg.Validate(); err != nil {
		return err
	}
	o.gcCfg = gcCfg
	o.fileSize = override.Numeric(1*telem.Gigabyte, o.fileSize)
	v := validate.New("cesium.options")
	v.Positive("chunk_size", o.chunkSize)
	return v.Error()
}

// WithFS sets the file system that cesium will use to store data. This defaults to the
// OS file system.
func WithFS(fs xfs.FS) Option { return func(o *options) { o.fs = fs } }

// WithGCConfig sets the garbage collection configuration for the DB. See the GCConfig
// struct for more details.
func WithGCConfig(gcCfg GCConfig) Option { return func(o *options) { o.gcCfg = gcCfg } }

// WithInstrumentation sets the instrumentation the DB will use for logging, tracing,
// etc. Defaults to noop instrumentation.
func WithInstrumentation(i alamos.Instrumentation) Option {
	return func(o *options) { o.Instrumentation = i }
}

// WithFileSizeCap sets the FileSize parameter of the database. FileSize is the maximum
// size, in bytes, for a writer to be created on a file. Note while that a file's size
// may still exceed this value, it is not likely to exceed by much with frequent
// commits. Defaults to 1GB.
func WithFileSizeCap(size telem.Size) Option {
	return func(o *options) { o.fileSize = size }
}

// WithChunkSize sets the most samples, summed over an iterator's channels, that one
// chunk of a fixed-span Next or Prev holds. A run of equal timestamps can exceed it.
// Defaults to 8,388,608 samples. IteratorConfig.AutoChunkSize sets the chunk size of
// an automatic span, per channel.
func WithChunkSize(size int64) Option {
	return func(o *options) { o.chunkSize = size }
}

// WithClock sets the clock that times the AutoIndexPersistInterval of each writer. It
// must be monotonic. Defaults to xtime.Real.
func WithClock(clock xtime.Clock) Option { return func(o *options) { o.clock = clock } }
