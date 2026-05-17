// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// expandKeysForAutoIndexing returns a config whose Keys include any index channels
// referenced by the data channels in cfg.Keys that are not already present, along with
// the set of indexes that were added implicitly. When per-channel authorities are
// supplied, each implicit index inherits the maximum authority of the data channels
// that reference it, since writing to a data channel requires successfully writing its
// index. When a single broadcast authority is supplied, the appended index keys
// naturally inherit it.
//
// channels must be the resolved channel metadata for cfg.Keys.
func expandKeysForAutoIndexing(
	cfg Config,
	channels []channel.Channel,
) (Config, set.Set[channel.Key]) {
	existing := set.New(cfg.Keys...)
	perChannelAuth := len(cfg.Authorities) > 1
	var keyAuth map[channel.Key]control.Authority
	if perChannelAuth {
		keyAuth = make(map[channel.Key]control.Authority, len(cfg.Keys))
		for i, k := range cfg.Keys {
			keyAuth[k] = cfg.Authorities[i]
		}
	}
	indexAuth := make(map[channel.Key]control.Authority)
	var implicit channel.Keys
	for _, ch := range channels {
		if ch.IsIndex {
			continue
		}
		idxKey := ch.Index()
		if idxKey == 0 || existing.Contains(idxKey) {
			continue
		}
		if _, seen := indexAuth[idxKey]; !seen {
			implicit = append(implicit, idxKey)
			indexAuth[idxKey] = 0
		}
		if perChannelAuth {
			if a := keyAuth[ch.Key()]; a > indexAuth[idxKey] {
				indexAuth[idxKey] = a
			}
		}
	}
	if len(implicit) == 0 {
		return cfg, nil
	}
	cfg.Keys = append(cfg.Keys, implicit...)
	if perChannelAuth {
		for _, idxKey := range implicit {
			cfg.Authorities = append(cfg.Authorities, indexAuth[idxKey])
		}
	}
	return cfg, set.New(implicit...)
}

// autoIndexer is a pipeline segment that injects server-generated timestamps for any
// index channel in the writer's keys whose series is omitted from an inbound Write
// frame, and propagates SetAuthority changes on data channels to the implicit index
// channels they reference. The segment is a no-op when disabled.
type autoIndexer struct {
	confluence.AbstractLinear[Request, Request]
	enabled   bool
	indexKeys channel.Keys
	// dataToIndex maps each non-index data channel key in the writer to the key of its
	// index channel. Used to size the timestamp series generated for each missing index
	// from the length of a data channel that references it, and to recompute implicit
	// index authorities when a referencing data channel's authority changes.
	dataToIndex map[channel.Key]channel.Key
	// implicitIndexes is the set of index channel keys that were added to the writer's
	// Keys by expandKeysForAutoIndexing. Only these indexes have their authority
	// recomputed when a SetAuthority call updates a referencing data channel.
	implicitIndexes set.Set[channel.Key]
	// dataAuth tracks the most recent authority observed for each data channel in the
	// writer, seeded from the open-time config and updated on every SetAuthority call.
	// Used to recompute implicit index authorities as the max across referencing data
	// channels.
	dataAuth      map[channel.Key]control.Authority
	highWaterMark telem.TimeStamp
}

func newAutoIndexer(
	enabled bool,
	channels []channel.Channel,
	keys channel.Keys,
	authorities []control.Authority,
	implicitIndexes set.Set[channel.Key],
	start telem.TimeStamp,
) *autoIndexer {
	dataToIndex := make(map[channel.Key]channel.Key, len(channels))
	seen := set.New[channel.Key]()
	var indexKeys channel.Keys
	for _, ch := range channels {
		if ch.IsIndex {
			continue
		}
		idx := ch.Index()
		if idx == 0 {
			continue
		}
		dataToIndex[ch.Key()] = idx
		if !seen.Contains(idx) {
			seen.Add(idx)
			indexKeys = append(indexKeys, idx)
		}
	}
	dataAuth := make(map[channel.Key]control.Authority, len(dataToIndex))
	if len(authorities) > 0 {
		for i, k := range keys {
			if _, isData := dataToIndex[k]; !isData {
				continue
			}
			dataAuth[k] = authorities[i%len(authorities)]
		}
	}
	return &autoIndexer{
		enabled:         enabled,
		indexKeys:       indexKeys,
		dataToIndex:     dataToIndex,
		implicitIndexes: implicitIndexes,
		dataAuth:        dataAuth,
		highWaterMark:   start,
	}
}

// Flow implements confluence.Flow.
func (a *autoIndexer) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(a.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-a.In.Outlet():
				if !ok {
					return nil
				}
				if a.enabled {
					switch req.Command {
					case CommandWrite:
						if len(a.indexKeys) > 0 {
							req.Frame = a.stamp(req.Frame)
						}
					case CommandSetAuthority:
						req = a.propagateAuthority(req)
					}
				}
				if err := signal.SendUnderContext(ctx, a.Out.Inlet(), req); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}

// propagateAuthority synchronizes the autoIndexer's per-data-channel authority tracking
// with the incoming SetAuthority request, and for per-channel calls augments req.Config
// with an updated authority for each implicit index whose referencing data channels'
// max may have changed. Broadcast calls (Config.Keys empty) are forwarded unchanged —
// cesium already applies the broadcast authority across every channel in the writer,
// including implicit indexes — but the tracked state is refreshed so the next
// per-channel call computes correctly. Implicit indexes the caller explicitly included
// in Config.Keys are left untouched.
func (a *autoIndexer) propagateAuthority(req Request) Request {
	cfg := req.Config
	if len(cfg.Authorities) == 0 {
		return req
	}
	if len(cfg.Keys) == 0 {
		for k := range a.dataAuth {
			a.dataAuth[k] = cfg.Authorities[0]
		}
		return req
	}

	if len(cfg.Authorities) == 1 {
		auth := cfg.Authorities[0]
		cfg.Authorities = make([]control.Authority, len(cfg.Keys))
		for i := range cfg.Authorities {
			cfg.Authorities[i] = auth
		}
	}
	for i, k := range cfg.Keys {
		if _, isData := a.dataToIndex[k]; isData {
			a.dataAuth[k] = cfg.Authorities[i]
		}
	}

	explicit := set.New(cfg.Keys...)
	for _, idxKey := range a.indexKeys {
		if !a.implicitIndexes.Contains(idxKey) || explicit.Contains(idxKey) {
			continue
		}
		var maxAuth control.Authority
		for dc, idx := range a.dataToIndex {
			if idx != idxKey {
				continue
			}
			if a.dataAuth[dc] > maxAuth {
				maxAuth = a.dataAuth[dc]
			}
		}
		cfg.Keys = append(cfg.Keys, idxKey)
		cfg.Authorities = append(cfg.Authorities, maxAuth)
	}

	req.Config = cfg
	return req
}

// stamp inspects fr, injects a TimeStamp series for each index channel in the writer's
// keys that is missing from fr, and advances the writer's high-water mark to cover the
// timestamps that were just generated. Each generated series is sized to match the
// length of a data channel that references its index, so two missing indexes covering
// data channels of different lengths in the same frame are stamped independently.
func (a *autoIndexer) stamp(fr frame.Frame) frame.Frame {
	present := make(set.Set[channel.Key])
	dataLens := make(map[channel.Key]int64)
	for i, k := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(i) {
			continue
		}
		present.Add(k)
		if idxKey, ok := a.dataToIndex[k]; ok {
			if _, recorded := dataLens[idxKey]; !recorded {
				dataLens[idxKey] = fr.RawSeriesAt(i).Len()
			}
		}
	}

	var maxLen int64
	for _, idxKey := range a.indexKeys {
		if present.Contains(idxKey) {
			continue
		}
		if n := dataLens[idxKey]; n > maxLen {
			maxLen = n
		}
	}
	if maxLen <= 0 {
		return fr
	}

	t0 := telem.Now()
	if a.highWaterMark > 0 && a.highWaterMark+1 > t0 {
		t0 = a.highWaterMark + 1
	}

	stamps := make([]telem.TimeStamp, maxLen)
	for j := range stamps {
		stamps[j] = t0 + telem.TimeStamp(j)
	}
	for _, idxKey := range a.indexKeys {
		if present.Contains(idxKey) {
			continue
		}
		n := dataLens[idxKey]
		if n == 0 {
			continue
		}
		fr = fr.Append(idxKey, telem.NewSeriesV(stamps[:n]...))
	}
	a.highWaterMark = t0 + telem.TimeStamp(maxLen-1)
	return fr
}
