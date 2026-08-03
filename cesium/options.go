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
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Option func(*options)

type options struct {
	alamos.Instrumentation
	fs               xfs.FS
	metaCodec        encoding.Codec
	dirname          string
	gcCfg            GCConfig
	fileSize         telem.Size
	streamBufferSize int
}

func (o *options) Report() alamos.Report { return alamos.Report{"dirname": o.dirname} }

func newOptions(dirname string, opts ...Option) (*options, error) {
	o := &options{dirname: dirname}
	for _, opt := range opts {
		opt(o)
	}
	if err := mergeAndValidateOptions(o); err != nil {
		return nil, err
	}
	return o, nil
}

func mergeAndValidateOptions(o *options) error {
	o.metaCodec = override.Nil[encoding.Codec](json.Codec, o.metaCodec)
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
	o.streamBufferSize = override.Numeric(defaultRelayBufferSize, o.streamBufferSize)
	v := validate.New("cesium.options")
	validate.Positive(v, "stream_buffer_size", o.streamBufferSize)
	return v.Error()
}

// WithFS sets the file system that cesium will use to store data. This defaults to the
// OS file system.
func WithFS(fs xfs.FS) Option { return func(o *options) { o.fs = fs } }

// WithGCConfig sets the garbage collection configuration for the DB. See the GCConfig
// struct for more details.
func WithGCConfig(config GCConfig) Option {
	return func(o *options) { o.gcCfg = config }
}

// WithInstrumentation sets the instrumentation the DB will use for logging, tracing,
// etc. Defaults to noop instrumentation.
func WithInstrumentation(i alamos.Instrumentation) Option {
	return func(o *options) { o.Instrumentation = i }
}

// WithFileSizeCap sets the FileSize parameter of the database. FileSize is the maximum
// size, in bytes, for a writer to be created on a file. Note while that a file's size
// may still exceed this value, it is not likely to exceed by much with frequent
// commits. Defaults to 1GB
func WithFileSizeCap(cap telem.Size) Option {
	return func(o *options) { o.fileSize = cap }
}

// WithStreamBufferSize sets the buffer size, in frames, of the relay pipe that streams
// written frames to streamers, and of each streamer's connection to that pipe. Defaults
// to 1000 frames.
func WithStreamBufferSize(size int) Option {
	return func(o *options) { o.streamBufferSize = size }
}
