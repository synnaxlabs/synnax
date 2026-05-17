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
// referenced by the data channels in cfg.Keys that are not already present. When
// per-channel authorities are supplied, each appended index inherits the maximum
// authority of the data channels that reference it, since writing to a data channel
// requires successfully writing its index. When a single broadcast authority is
// supplied, the appended index keys naturally inherit it.
//
// channels must be the resolved channel metadata for cfg.Keys.
func expandKeysForAutoIndexing(cfg Config, channels []channel.Channel) Config {
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
		return cfg
	}
	cfg.Keys = append(cfg.Keys, implicit...)
	if perChannelAuth {
		for _, idxKey := range implicit {
			cfg.Authorities = append(cfg.Authorities, indexAuth[idxKey])
		}
	}
	return cfg
}

// autoIndexer runs on each leaseholder's storage-writer-facing pipeline and provides
// two services scoped to channels local to that leaseholder:
//
//   - On CommandWrite, it injects a TimeStamp series for each index channel in the
//     writer's keys whose series is omitted from the inbound frame. Timestamps are
//     produced by this node's clock so the index data is stamped by the same node
//     that persists it.
//   - On CommandSetAuthority, it propagates authority changes on data channels to the
//     index channels they reference, taking the max across referencing data channels.
//     If the SetAuthority call explicitly includes an index in its Keys, that index is
//     left untouched — the caller's explicit value wins.
//
// The segment is a no-op when disabled or when there are no index channels in scope.
type autoIndexer struct {
	confluence.AbstractLinear[Request, Request]
	// indexKeys lists every leaseholder-local index channel that has at least one
	// referencing data channel also in the writer's local keys.
	indexKeys channel.Keys
	// dataToIndex maps each leaseholder-local non-index data channel in the writer's
	// keys to the key of its index channel.
	dataToIndex map[channel.Key]channel.Key
	// dataAuth tracks the most recent authority observed for each local data channel,
	// seeded from the open-time config and updated on every SetAuthority call. Used to
	// recompute index authorities as the max across referencing data channels.
	dataAuth      map[channel.Key]control.Authority
	highWaterMark telem.TimeStamp
}

// newAutoIndexer constructs an autoIndexer scoped to channels — the channel metadata
// for the leaseholder's local key slice. keys and authorities are the writer's
// leaseholder-local open-time configuration, used to seed the dataAuth map. start
// seeds the high-water mark so the first auto-stamped sample is >= start + 1.
func newAutoIndexer(
	channels []channel.Channel,
	keys channel.Keys,
	authorities []control.Authority,
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
		indexKeys:     indexKeys,
		dataToIndex:   dataToIndex,
		dataAuth:      dataAuth,
		highWaterMark: start,
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
				switch req.Command {
				case CommandWrite:
					if len(a.indexKeys) > 0 {
						req.Frame = a.stamp(req.Frame)
					}
				case CommandSetAuthority:
					req = a.propagateAuthority(req)
				}
				if err := signal.SendUnderContext(ctx, a.Out.Inlet(), req); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}

// propagateAuthority synchronizes the autoIndexer's per-data-channel authority tracking
// with the incoming SetAuthority request and augments req.Config with an updated
// authority for each local index whose referencing data channels' max may have changed.
// Broadcast calls (Config.Keys empty) are forwarded unchanged — cesium already applies
// the broadcast authority across every channel in the writer, including indexes — but
// the tracked state is refreshed so the next per-channel call computes correctly.
// Indexes the caller explicitly included in Config.Keys are left untouched.
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
		if explicit.Contains(idxKey) {
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

// stamp inspects fr, injects a TimeStamp series for each local index channel that is
// missing from fr, and advances the high-water mark to cover the timestamps that were
// just generated. Each generated series is sized to match the length of a data channel
// that references its index, so two missing indexes covering data channels of different
// lengths in the same frame are stamped independently.
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
