// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package doctor inspects the data a Core stored on disk without opening it for
// service. Every check is strictly read-only: the doctor decodes files and reads the
// key-value store in read-only mode, and never migrates, compacts, or repairs.
package doctor

import (
	"context"
	"maps"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/inspect"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/storage"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Phase names one stage of a run, reported through Config.Progress.
type Phase string

const (
	// PhaseTS inspects the time-series store.
	PhaseTS Phase = "time-series"
	// PhaseKV inspects the key-value store.
	PhaseKV Phase = "key-value"
)

// Config configures a doctor run.
type Config struct {
	alamos.Instrumentation
	// Dirname is the Core data directory to inspect.
	//
	// [REQUIRED]
	Dirname string
	// FS is the file system the data directory lives on.
	//
	// [OPTIONAL] - Defaults to the OS file system.
	FS xfs.FS
	// Codec decodes stored key-value entries.
	//
	// [OPTIONAL] - Defaults to the chain the Core writes with.
	Codec encoding.Codec
	// Deep enables the checks that read sample bytes.
	//
	// [OPTIONAL] - Defaults to true.
	Deep *bool
	// TSDisabled skips every time-series check.
	//
	// [OPTIONAL] - Defaults to false.
	TSDisabled bool
	// KVDisabled skips every key-value check.
	//
	// [OPTIONAL] - Defaults to false.
	KVDisabled bool
	// Channels restricts the time-series inspection to the given channels. Empty
	// inspects every channel.
	//
	// [OPTIONAL]
	Channels []channel.Key
	// Progress reports how far through a phase the run is.
	//
	// [OPTIONAL] - Defaults to no reporting.
	Progress func(phase Phase, done, total int)
}

var (
	_ xconfig.Config[Config] = Config{}
	// DefaultConfig holds the defaults every run starts from.
	DefaultConfig = Config{
		FS:    xfs.Default,
		Codec: encoding.NewDecodeFallbackCodec(orc.Codec, msgpack.Codec),
		Deep:  new(true),
	}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Dirname = override.String(c.Dirname, other.Dirname)
	c.FS = override.Nil(c.FS, other.FS)
	c.Codec = override.Nil(c.Codec, other.Codec)
	c.Deep = override.Nil(c.Deep, other.Deep)
	c.TSDisabled = c.TSDisabled || other.TSDisabled
	c.KVDisabled = c.KVDisabled || other.KVDisabled
	c.Channels = override.Slice(c.Channels, other.Channels)
	c.Progress = override.Nil(c.Progress, other.Progress)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("doctor")
	validate.NotEmptyString(v, "dirname", c.Dirname)
	validate.NotNil(v, "fs", c.FS)
	validate.NotNil(v, "codec", c.Codec)
	return v.Error()
}

// Report is the result of a doctor run.
type Report struct {
	// Dirname is the data directory the run inspected.
	Dirname string `json:"dirname"`
	// TS holds the time-series report. Nil when the phase was skipped.
	TS *inspect.Report `json:"ts,omitempty"`
	// KV holds the key-value report. Nil when the phase was skipped or the store
	// could not be opened.
	KV *KVReport `json:"kv,omitempty"`
	// KVUnavailable states why the key-value store could not be read. Empty when it
	// was read, or when the phase was skipped.
	KVUnavailable string `json:"kv_unavailable,omitempty"`
	// Findings holds every finding the run produced, across both stores. TS holds
	// the same time-series findings grouped by channel.
	Findings []Finding `json:"findings"`
}

// Errors returns the number of error-level findings the run produced.
func (r Report) Errors() int {
	count := 0
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			count++
		}
	}
	return count
}

// fromTS converts a time-series finding into a doctor finding. The channel key leads
// the subject, so a flat finding still names what it is about.
func fromTS(f inspect.Finding) Finding {
	subject := f.Subject
	if f.Channel != 0 {
		subject = strconv.FormatUint(uint64(f.Channel), 10)
		if f.Subject != "" {
			subject += "/" + f.Subject
		}
	}
	return Finding{
		Check:    Check(f.Check),
		Severity: f.Severity,
		Subject:  subject,
		Message:  f.Message,
		Hint:     f.Hint,
	}
}

// tsFindings flattens every finding in a time-series report.
func tsFindings(r *inspect.Report) []Finding {
	findings := make([]Finding, 0, len(r.Findings))
	for _, f := range r.Findings {
		findings = append(findings, fromTS(f))
	}
	for _, ch := range r.Channels {
		for _, f := range ch.Findings {
			findings = append(findings, fromTS(f))
		}
	}
	return findings
}

// Run inspects the data directory named by the configuration. It returns an error only
// when the run itself could not complete; problems with the stored data are reported as
// findings. A key-value store another process holds open is reported as unavailable,
// and the time-series checks still run.
func Run(ctx context.Context, cfgs ...Config) (r Report, err error) {
	cfg, err := xconfig.New(DefaultConfig, cfgs...)
	if err != nil {
		return Report{}, err
	}
	exists, err := cfg.FS.Exists(cfg.Dirname)
	if err != nil {
		return Report{}, err
	}
	if !exists {
		return Report{}, errors.Wrapf(
			validate.ErrValidation, "data directory %s does not exist", cfg.Dirname,
		)
	}
	r = Report{Dirname: cfg.Dirname}
	var (
		db     kv.DB
		tables []table
	)
	if !cfg.KVDisabled {
		if db, err = openKV(cfg); err != nil {
			r.KVUnavailable = err.Error()
			db, err = nil, nil
		} else {
			defer func() { err = errors.Combine(err, db.Close()) }()
			tables = slices.Concat(newRegistry(), newConfigRegistry())
		}
	}
	if !cfg.TSDisabled {
		if r.TS, err = runTS(ctx, cfg); err != nil {
			return r, err
		}
		r.Findings = tsFindings(r.TS)
	}
	if db != nil {
		if err = runKV(ctx, cfg, db, tables, &r); err != nil {
			return r, err
		}
	}
	return r, err
}

// openKV opens the data directory's key-value store in read-only mode.
func openKV(cfg Config) (kv.DB, error) {
	return pebblekv.OpenReadOnly(
		filepath.Join(cfg.Dirname, storage.KVDirName),
		cfg.Instrumentation,
	)
}

// runKV runs every key-value, referential, and cross-layer check.
func runKV(
	ctx context.Context,
	cfg Config,
	db kv.DB,
	tables []table,
	r *Report,
) error {
	names := make([]string, len(tables))
	for i, t := range tables {
		names[i] = t.name
	}
	res, err := walk(db, names)
	if err != nil {
		return err
	}
	cluster, err := readCluster(ctx, db)
	if err != nil {
		return err
	}
	kvr := &KVReport{Cluster: cluster, Entries: res.entries, Bytes: res.bytes}
	for _, b := range res.buckets {
		kvr.Buckets = append(kvr.Buckets, *b)
	}
	slices.SortFunc(
		kvr.Buckets,
		func(a, b Bucket) int { return int(b.Bytes - a.Bytes) },
	)
	r.KV = kvr
	s := newState(db, cfg.Codec)
	for _, t := range tables {
		s.declare(t.ontologyType)
	}
	s.reportDecode = true
	for i, t := range tables {
		if err = t.collect(ctx, s); err != nil {
			return err
		}
		progress(cfg, PhaseKV, i+1, len(tables)*2)
	}
	s.reportDecode = false
	for i, t := range tables {
		if t.check == nil {
			continue
		}
		if err = t.check(ctx, s); err != nil {
			return err
		}
		progress(cfg, PhaseKV, len(tables)+i+1, len(tables)*2)
	}
	checkWalk(s, res)
	checkCounters(s, res, cluster.HostKey)
	if err = checkMigrations(ctx, s, tables, res); err != nil {
		return err
	}
	if r.TS != nil {
		checkCross(s, r.TS, cluster.HostKey)
	}
	r.Findings = append(r.Findings, s.findings()...)
	return nil
}

// progress reports one step of a phase when the configuration asked for it.
func progress(cfg Config, phase Phase, done, total int) {
	if cfg.Progress != nil {
		cfg.Progress(phase, done, total)
	}
}

// checkWalk reports what the structural walk found outside the known tables.
func checkWalk(s *state, res walkResult) {
	if res.unknown.count > 0 {
		s.noteN(
			CheckUnknownPrefix,
			"key belongs to no known table",
			res.unknown.first,
			res.unknown.count,
		)
	}
	if res.staging > 0 {
		s.noteN(
			CheckStaging,
			"task migration staging entries remain",
			bucketStaging,
			int(res.staging),
		)
	}
	for _, name := range slices.Sorted(maps.Keys(s.decode)) {
		v := s.decode[name]
		s.noteN(
			CheckDecode,
			"entry in "+name+" cannot be decoded",
			v.first,
			v.count,
		)
	}
}

// checkMigrations reports tables holding entries whose applied migrations are behind
// the chain this binary ships.
func checkMigrations(
	ctx context.Context,
	s *state,
	tables []table,
	res walkResult,
) error {
	db := gorp.Wrap(s.db, gorp.WithCodec(s.codec))
	for _, t := range tables {
		bucket, found := res.buckets[t.name]
		if len(t.migrations) == 0 || !found || bucket.Entries == 0 {
			continue
		}
		applied, err := gorp.AppliedMigrations(ctx, db, t.name)
		if err != nil {
			return err
		}
		for _, m := range t.migrations {
			if !applied.Contains(m.Key()) {
				s.note(
					CheckMigrationState,
					"table has not applied every migration",
					t.name+": "+m.Key(),
				)
			}
		}
	}
	return nil
}
