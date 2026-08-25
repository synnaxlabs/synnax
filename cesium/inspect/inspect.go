// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package inspect reports on the health of a Cesium database by decoding its files
// directly. It is strictly read-only: it never opens the engine, never migrates
// metadata, and never repairs anything, so it is safe to run on suspect data and,
// with known races, on a database a live engine is serving.
package inspect

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for running an inspection.
type Config struct {
	alamos.Instrumentation
	// FS is the file system rooted at the Cesium data directory.
	// [REQUIRED]
	FS fs.FS
	// Deep enables checks that read sample bytes. Their cost scales with stored data
	// size instead of domain count.
	// [OPTIONAL] Default: true
	Deep *bool
	// Channels restricts full inspection to the given keys. Metadata for every
	// channel is still read so cross-channel checks stay accurate.
	// [OPTIONAL] Default: all channels
	Channels []channel.Key
	// FileSize mirrors domain.Config.FileSize for GC-eligibility computation.
	// [OPTIONAL] Default: 800 MB
	FileSize telem.Size
	// GCThreshold mirrors domain.Config.GCThreshold. Must be in [0, 1].
	// [OPTIONAL] Default: 0.2
	GCThreshold float32
	// TinyDomainSize is the byte floor below which a domain counts as tiny.
	// [OPTIONAL] Default: 16 bytes
	TinyDomainSize telem.Size
	// TinyDomainSpan is the span floor below which a domain counts as tiny.
	// [OPTIONAL] Default: 1 ms
	TinyDomainSpan telem.TimeSpan
	// MicroGapMax is the upper bound of the near-zero gap band: gaps in
	// (0, MicroGapMax] count as micro-gaps.
	// [OPTIONAL] Default: 1 ms
	MicroGapMax telem.TimeSpan
	// FarPast marks domain bounds before it as implausible.
	// [OPTIONAL] Default: 2000-01-01
	FarPast telem.TimeStamp
	// NearEpochMax bounds the band [0, NearEpochMax] whose timestamps carry the
	// elapsed-time-written-as-timestamp signature.
	// [OPTIONAL] Default: one year after the Unix epoch
	NearEpochMax telem.TimeStamp
	// FarFutureSlack marks domain bounds after now plus the slack as implausible.
	// [OPTIONAL] Default: 24 h
	FarFutureSlack telem.TimeSpan
	// Now is the wall-clock source for far-future detection.
	// [OPTIONAL] Default: telem.Now
	Now func() telem.TimeStamp
	// Progress, when set, is called after each channel completes.
	// [OPTIONAL]
	Progress func(done, total int)
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for an inspection.
	DefaultConfig = Config{
		Deep:           new(true),
		FileSize:       800 * telem.Megabyte,
		GCThreshold:    0.2,
		TinyDomainSize: 16 * telem.Byte,
		TinyDomainSpan: telem.Millisecond,
		MicroGapMax:    telem.Millisecond,
		FarPast: telem.NewTimeStamp(
			time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		),
		NearEpochMax: telem.NewTimeStamp(
			time.Date(1971, 1, 1, 0, 0, 0, 0, time.UTC),
		),
		FarFutureSlack: 24 * telem.Hour,
		Now:            telem.Now,
	}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.FS = override.Nil(c.FS, other.FS)
	c.Deep = override.Nil(c.Deep, other.Deep)
	c.Channels = override.Slice(c.Channels, other.Channels)
	c.FileSize = override.Numeric(c.FileSize, other.FileSize)
	c.GCThreshold = override.Numeric(c.GCThreshold, other.GCThreshold)
	c.TinyDomainSize = override.Numeric(c.TinyDomainSize, other.TinyDomainSize)
	c.TinyDomainSpan = override.Numeric(c.TinyDomainSpan, other.TinyDomainSpan)
	c.MicroGapMax = override.Numeric(c.MicroGapMax, other.MicroGapMax)
	c.FarPast = override.Numeric(c.FarPast, other.FarPast)
	c.NearEpochMax = override.Numeric(c.NearEpochMax, other.NearEpochMax)
	c.FarFutureSlack = override.Numeric(c.FarFutureSlack, other.FarFutureSlack)
	c.Now = override.Nil(c.Now, other.Now)
	c.Progress = override.Nil(c.Progress, other.Progress)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("inspect")
	validate.NotNil(v, "fs", c.FS)
	validate.Positive(v, "file_size", c.FileSize)
	validate.GreaterThanEq(v, "gc_threshold", c.GCThreshold, 0)
	validate.LessThanEq(v, "gc_threshold", c.GCThreshold, 1)
	return v.Error()
}

// deleteArtifactInfix marks directories renamed by an interrupted channel delete.
const deleteArtifactInfix = "-DELETE-"

// Run inspects the Cesium database rooted at the configured file system and returns a
// report of per-channel statistics and findings. It returns an error only when the
// inspection itself cannot run; problems with the data are reported as findings.
func Run(ctx context.Context, cfgs ...Config) (Report, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return Report{}, err
	}
	entries, err := cfg.FS.List("")
	if err != nil {
		return Report{}, err
	}
	var (
		report Report
		dirs   = make(map[channel.Key]string, len(entries))
	)
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() {
			report.Findings = append(report.Findings, newFinding(
				CheckOrphanFile, 0, name,
				"unrecognized file in the database root",
			))
			continue
		}
		if strings.Contains(name, deleteArtifactInfix) {
			report.Findings = append(report.Findings, newFinding(
				CheckArtifacts, 0, name,
				"directory left behind by an interrupted channel delete",
			))
			continue
		}
		key, err := strconv.Atoi(name)
		if err != nil {
			report.Findings = append(report.Findings, newFinding(
				CheckOrphanFile, 0, name,
				"directory name is not a channel key",
			))
			continue
		}
		dirs[channel.Key(key)] = name
	}

	// Metadata is read for every channel regardless of the filter so that
	// cross-channel checks (index references) see the whole database.
	metas := make(map[channel.Key]metaResult, len(dirs))
	for key, name := range dirs {
		metas[key] = readMeta(ctx, cfg.FS, name)
	}

	keys := make([]channel.Key, 0, len(dirs))
	filter := set.New(cfg.Channels...)
	for key := range dirs {
		if len(cfg.Channels) == 0 || filter.Contains(key) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)

	for i, key := range keys {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		sub, err := cfg.FS.Sub(dirs[key])
		if err != nil {
			return Report{}, err
		}
		chReport, err := inspectChannel(ctx, cfg, key, sub, metas)
		if err != nil {
			return Report{}, err
		}
		report.Channels = append(report.Channels, chReport)
		if cfg.Progress != nil {
			cfg.Progress(i+1, len(keys))
		}
	}
	report.aggregate()
	return report, nil
}

// metaResult is the outcome of decoding one channel's meta.json.
type metaResult struct {
	channel channel.Channel
	err     error
}

// readMeta decodes the meta.json inside the named channel directory without running
// migrations or rewriting anything.
func readMeta(ctx context.Context, dataFS fs.FS, name string) metaResult {
	sub, err := dataFS.Sub(name)
	if err != nil {
		return metaResult{err: err}
	}
	ch, err := meta.Read(ctx, sub, json.Codec)
	return metaResult{channel: ch, err: err}
}
