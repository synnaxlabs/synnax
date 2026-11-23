// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

import (
	"context"
	"encoding/json/v2"
	"unsafe"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ActionConfig configures an action channel. Each sample is the JSON of one
// actions.Scoped batch.
type ActionConfig[K gorp.Key, A any] struct {
	// Name is the name of the channel.
	//
	// [REQUIRED]
	Name string
	// Actions holds the batches that the transactions of the service attach.
	//
	// [REQUIRED]
	Actions *actions.State[K, A]
	// Resource returns the resource of the entry with the given key. A streamer sends a
	// batch only to a subject that can retrieve the entry that it edits.
	//
	// [REQUIRED]
	Resource func(K) resource.ID
	// Table names a table that only the channel describes. Each create or update of one
	// of its records needs a batch for its key in the same commit, and the Provider
	// reports a change without one with DPanic.
	//
	// [OPTIONAL] - Defaults to no table.
	Table string
	// BatchChecks returns the lookup of the check of each batch that the transaction of
	// c attached, by the key of its entry and its number. It runs once for each commit
	// with batches. A streamer sends a batch only to a subject that passes its check,
	// in place of a check of Resource. A nil check leaves the batch to Resource.
	//
	// [OPTIONAL] - Defaults to a check of Resource.
	BatchChecks func(c gorp.Commit) func(key K, seq uint64) (check, bool)
}

// Validate checks that the required fields of c are set.
func (c ActionConfig[K, A]) Validate() error {
	v := validate.New("signals.action_config")
	v.NotEmptyString("name", c.Name)
	v.NotNil("actions", c.Actions)
	v.NotNil("resource", c.Resource)
	return v.Error()
}

// PublishActions creates an action channel and adds it to the channels that Start
// writes. Call it before Start.
func (p *Provider) PublishActions[K gorp.Key, A any](
	ctx context.Context,
	cfg ActionConfig[K, A],
) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	f := &actionFeed[K, A]{
		cfg:    cfg,
		L:      p.cfg.L,
		ch:     channel.Channel{Name: cfg.Name, DataType: telem.JSONT},
		checks: newCommitChecks(p.checkCommits),
	}
	return p.add(ctx, f, &f.ch)
}

// actionFeed writes the samples of one action channel.
type actionFeed[K gorp.Key, A any] struct {
	cfg ActionConfig[K, A]
	L   *alamos.Logger
	ch  channel.Channel
	// checks holds the check of each batch, by the bytes of its sample.
	checks *commitChecks
}

func (f *actionFeed[K, A]) append(c gorp.Commit, fr framer.Frame) framer.Frame {
	batches := f.cfg.Actions.Of(c)
	if f.cfg.Table != "" {
		f.check(c, batches)
	}
	samples := telem.Series{DataType: telem.JSONT}
	var (
		checks map[string]check
		lookup func(K, uint64) (check, bool)
	)
	if f.cfg.BatchChecks != nil && len(batches) > 0 {
		checks = make(map[string]check, len(batches))
		lookup = f.cfg.BatchChecks(c)
		defer func() { f.checks.add(checks) }()
	}
	for _, batch := range batches {
		b, err := json.Marshal(batch)
		if err != nil {
			f.L.Error(
				"failed to marshal an action batch",
				zap.String("channel", f.ch.Name),
				zap.Error(err),
			)
			continue
		}
		if checks != nil {
			check, ok := lookup(batch.Key, batch.Seq)
			if !ok {
				f.L.DPanic(
					"an action batch has no check",
					zap.String("channel", f.ch.Name),
					zap.Uint64("seq", batch.Seq),
				)
				continue
			}
			// Nothing writes b again, so the key shares its bytes, as add requires.
			checks[unsafe.String(unsafe.SliceData(b), len(b))] = check
		}
		samples.Data = append(samples.Data, telem.MarshalVariableSample(b)...)
	}
	if len(samples.Data) == 0 {
		return fr
	}
	return fr.Append(f.ch.Key(), samples)
}

func (f *actionFeed[K, A]) records() map[channel.Key]Record {
	r := Record{Of: recordOf(unmarshalKey[K], f.cfg.Resource)}
	if f.cfg.BatchChecks != nil {
		r.Check = func(sample []byte) (check, error) { return f.checks.get(sample) }
	}
	return map[channel.Key]Record{f.ch.Key(): r}
}

// check reports each create or update in c of a record of the table of f that has no
// batch for its key.
func (f *actionFeed[K, A]) check(c gorp.Commit, batches []actions.Scoped[K, A]) {
	covered := make(set.Set[string], len(batches))
	for _, b := range batches {
		covered.Add(gorp.FormatKey(b.Key))
	}
	for _, ch := range c.Changes {
		if ch.Record.Table != f.cfg.Table || ch.Kind == gorp.ChangeDelete ||
			covered.Contains(ch.Record.Key) {
			continue
		}
		f.L.DPanic(
			"a change has no action batch",
			zap.String("channel", f.ch.Name),
			zap.String("table", ch.Record.Table),
			zap.String("key", ch.Record.Key),
		)
	}
}
