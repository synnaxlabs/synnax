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
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
)

// withIndexes returns cfg with the index of each data channel in cfg.Keys that cfg
// lacks. indexes maps each data channel to its index. An added index takes the highest
// authority of the data channels it indexes, since a write to a data channel needs a
// write to its index. cfg must hold one authority per key.
func withIndexes(cfg Config, indexes map[cesium.ChannelKey]cesium.ChannelKey) Config {
	var (
		n        = len(cfg.Keys)
		existing = set.New(cfg.Keys...)
		added    map[channel.Key]int
	)
	for i := range n {
		idx, ok := indexes[uint32(cfg.Keys[i])]
		if !ok || existing.Contains(channel.Key(idx)) {
			continue
		}
		if j, ok := added[channel.Key(idx)]; ok {
			cfg.Authorities[j] = max(cfg.Authorities[j], cfg.Authorities[i])
			continue
		}
		if added == nil {
			added = make(map[channel.Key]int)
		}
		added[channel.Key(idx)] = len(cfg.Keys)
		cfg.Keys = append(cfg.Keys, channel.Key(idx))
		cfg.Authorities = append(cfg.Authorities, cfg.Authorities[i])
	}
	return cfg
}

// indexStamper adds a series of host timestamps for each written index that a frame
// lacks while it holds a data channel of that index. A series spaces its stamps 1ns
// apart and starts after the last stamp of its index, so the stamps of an index rise
// across frames. It is not safe for concurrent use.
type indexStamper struct {
	groups []*stampGroup
	// groupOf maps each key of a group, its index included, to the group.
	groupOf map[cesium.ChannelKey]*stampGroup
}

// stampGroup is a written index and the written data channels it indexes.
type stampGroup struct {
	index cesium.ChannelKey
	// clock is the last stamp of the index.
	clock telem.TimeStamp
	// indexPresent and dataLen hold the scan of the current frame: whether the frame
	// holds the index, and the length of its first data series that is not empty.
	indexPresent bool
	dataLen      int64
}

// newIndexStamper returns a stamper for the indexes in keys of the data channels in
// keys. indexes maps each data channel to its index. The first stamp of each index is
// after start.
func newIndexStamper(
	keys []cesium.ChannelKey,
	indexes map[cesium.ChannelKey]cesium.ChannelKey,
	start telem.TimeStamp,
) *indexStamper {
	written := set.New(keys...)
	st := &indexStamper{groupOf: make(map[cesium.ChannelKey]*stampGroup)}
	for _, k := range keys {
		idx, ok := indexes[k]
		if !ok || !written.Contains(idx) {
			continue
		}
		g, ok := st.groupOf[idx]
		if !ok {
			g = &stampGroup{index: idx, clock: start}
			st.groups = append(st.groups, g)
			st.groupOf[idx] = g
		}
		st.groupOf[k] = g
	}
	return st
}

// stamp returns fr with the series of each index it lacks.
func (st *indexStamper) stamp(fr cesium.Frame) cesium.Frame {
	for _, g := range st.groups {
		g.indexPresent, g.dataLen = false, 0
	}
	for i, k := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(i) {
			continue
		}
		g, ok := st.groupOf[k]
		if !ok {
			continue
		}
		if k == g.index {
			g.indexPresent = true
		} else if g.dataLen == 0 {
			g.dataLen = fr.RawSeriesAt(i).Len()
		}
	}
	var toAdd int
	for _, g := range st.groups {
		if g.stamped() {
			toAdd++
		}
	}
	if toAdd == 0 {
		return fr
	}
	fr = fr.Grow(toAdd)
	now := telem.Now()
	for _, g := range st.groups {
		if g.stamped() {
			fr = fr.Append(g.index, g.next(now))
		}
	}
	return fr
}

func (g *stampGroup) stamped() bool { return !g.indexPresent && g.dataLen > 0 }

// next returns the next dataLen stamps of the index, from now or after its last stamp.
func (g *stampGroup) next(now telem.TimeStamp) telem.Series {
	t0 := max(now, g.clock+1)
	// One allocation: the stamps are written into the series buffer directly.
	series := telem.MakeSeries(telem.TimestampT, int(g.dataLen))
	stamps := unsafe.CastSlice[byte, telem.TimeStamp](series.Data)
	for j := range stamps {
		stamps[j] = t0 + telem.TimeStamp(j)
	}
	g.clock = stamps[len(stamps)-1]
	return series
}
