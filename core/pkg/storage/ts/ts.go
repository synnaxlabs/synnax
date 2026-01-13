// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ts

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/config"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type (
	DB               = cesium.DB
	Frame            = cesium.Frame
	Channel          = cesium.Channel
	ChannelKey       = cesium.ChannelKey
	WriterConfig     = cesium.WriterConfig
	Writer           = cesium.Writer
	WriterMode       = cesium.WriterMode
	StreamWriter     = cesium.StreamWriter
	WriterRequest    = cesium.WriterRequest
	WriterResponse   = cesium.WriterResponse
	WriterCommand    = cesium.WriterCommand
	ControlDigest    = cesium.ControlUpdate
	IteratorConfig   = cesium.IteratorConfig
	Iterator         = cesium.Iterator
	StreamIterator   = cesium.StreamIterator
	IteratorRequest  = cesium.IteratorRequest
	IteratorResponse = cesium.IteratorResponse
	IteratorCommand  = cesium.IteratorCommand
	StreamerConfig   = cesium.StreamerConfig
	StreamerRequest  = cesium.StreamerRequest
	StreamerResponse = cesium.StreamerResponse
)

const AutoSpan = cesium.AutoSpan
const (
	WriterPersistStream = cesium.WriterModePersistStream
	WriterPersistOnly   = cesium.WriterModePersistOnly
	WriterStreamOnly    = cesium.WriterModeStreamOnly
)

type Config struct {
	alamos.Instrumentation
	// Dirname is the directory in which the DB will store its files. DB should have
	// exclusive access to this directory, and sharing with another process/component
	// may result in data corruption.
	//
	// [OPTIONAL] - Defaults to an empty string.
	Dirname string
	// FS is the file system interface that the DB will use to read and write data.
	//
	// [REQUIRED]
	FS xfs.FS
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for a DB.
	DefaultConfig = Config{
		FS: xfs.Default,
	}
	ErrChannelNotfound = cesium.ErrChannelNotFound
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("ts")
	validate.NotNil(v, "fs", c.FS)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Dirname = override.String(c.Dirname, other.Dirname)
	c.FS = override.Nil(c.FS, other.FS)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

func Open(ctx context.Context, configs ...Config) (*DB, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	return cesium.Open(
		ctx,
		cfg.Dirname,
		cesium.WithFS(cfg.FS),
		cesium.WithInstrumentation(cfg.Instrumentation),
	)
}
