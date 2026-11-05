// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package gop keeps the frames of each video channel since its last keyframe, so a new
// viewer can start to decode without waiting for the next keyframe.
package gop

import (
	"context"
	"io"
	"maps"
	"slices"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Config is the configuration for opening a Cache.
type Config struct {
	alamos.Instrumentation
	// Channel finds the video channels and reports changes to them.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer streams the frames of the video channels.
	//
	// [REQUIRED]
	Framer *framer.Service
	// MaxFrames bounds the frames that the Cache holds for one channel. A longer GOP
	// empties the channel's entry until its next keyframe.
	//
	// [OPTIONAL] - Defaults to 600.
	MaxFrames int
	// MaxSize bounds the size of the frames that the Cache holds for one channel. A
	// larger GOP empties the channel's entry until its next keyframe.
	//
	// [OPTIONAL] - Defaults to 32 MB.
	MaxSize telem.Size
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for a Cache.
	DefaultConfig = Config{MaxFrames: 600, MaxSize: 32 * telem.Megabyte}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.MaxFrames = override.Numeric(c.MaxFrames, other.MaxFrames)
	c.MaxSize = override.Numeric(c.MaxSize, other.MaxSize)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("gop")
	v.NotNil("channel", c.Channel)
	v.NotNil("framer", c.Framer)
	v.Positive("max_frames", c.MaxFrames)
	v.Positive("max_size", c.MaxSize)
	return v.Error()
}

// Cache keeps the frames of each video channel since its last keyframe. It is safe
// for concurrent use.
type Cache struct {
	shutdown   io.Closer
	requests   confluence.Inlet[framer.StreamerRequest]
	disconnect observe.Disconnect
	cfg        Config
	// sendMu orders the key updates that the Cache sends to its streamer.
	sendMu sync.Mutex
	mu     struct {
		entries map[channel.Key]*entry
		sync.Mutex
	}
}

type entry struct {
	series []telem.Series
	end    telem.Alignment
	size   telem.Size
	frames int
}

// Continues returns true when a sample at next follows a sample that ended at end
// with no frame lost between them. A new domain counts as a continuation, because a
// writer moves to a new domain without losing samples.
func Continues(end, next telem.Alignment) bool {
	return next == end || next.DomainIndex() != end.DomainIndex()
}

func (e *entry) add(s telem.Series, cfg Config) {
	last := -1
	for i := range s.Keyframes() {
		last = i
	}
	if last >= 0 {
		*e = entry{}
		s = s.Slice(last, int(s.Len()))
	} else if len(e.series) == 0 || !Continues(e.end, s.Alignment) {
		*e = entry{}
		return
	}
	e.series = append(e.series, s)
	e.frames += int(s.Len())
	e.size += s.Size()
	e.end = s.AlignmentBounds().Upper
	if e.frames > cfg.MaxFrames || e.size > cfg.MaxSize {
		*e = entry{}
	}
}

// Open opens a Cache that follows every video channel. Close it to stop.
func Open(ctx context.Context, cfgs ...Config) (*Cache, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	streamer, err := cfg.Framer.NewStreamer(framer.StreamerConfig{})
	if err != nil {
		return nil, err
	}
	c := &Cache{cfg: cfg}
	c.mu.entries = make(map[channel.Key]*entry)
	requests, responses := confluence.Attach(streamer, 10)
	c.requests = requests
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	c.shutdown = signal.NewGracefulShutdown(sCtx, cancel)
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	sCtx.Go(func(context.Context) error {
		for res := range responses.Outlet() {
			c.add(res.Frame)
		}
		return nil
	}, signal.WithKey("gop"))
	c.disconnect = cfg.Channel.Observe().OnChange(c.handleChange)
	var channels []channel.Channel
	if err = cfg.Channel.NewRetrieve().
		Where(channel.Match(func(_ gorp.Context, _ channel.Retrieve, ch *channel.Channel) (bool, error) {
			return ch.DataType == telem.VideoT, nil
		})).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, errors.Combine(err, c.Close())
	}
	c.follow(ctx, func(entries map[channel.Key]*entry) bool {
		for _, ch := range channels {
			entries[ch.Key()] = &entry{}
		}
		return len(channels) > 0
	})
	return c, nil
}

func (c *Cache) handleChange(
	ctx context.Context,
	changes gorp.TxReader[channel.Key, channel.Channel],
) {
	c.follow(ctx, func(entries map[channel.Key]*entry) (changed bool) {
		for ch := range changes {
			_, followed := entries[ch.Key]
			switch {
			case ch.Variant == change.VariantSet && !followed &&
				ch.Value.DataType == telem.VideoT:
				entries[ch.Key] = &entry{}
				changed = true
			case ch.Variant == change.VariantDelete && followed:
				delete(entries, ch.Key)
				changed = true
			}
		}
		return changed
	})
}

// follow applies update to the followed channels, and streams the new set when update
// returns true.
func (c *Cache) follow(
	ctx context.Context,
	update func(map[channel.Key]*entry) bool,
) {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	c.mu.Lock()
	changed := update(c.mu.entries)
	keys := slices.Collect(maps.Keys(c.mu.entries))
	c.mu.Unlock()
	if !changed {
		return
	}
	if err := signal.SendUnderContext(
		ctx,
		c.requests.Inlet(),
		framer.StreamerRequest{Keys: keys},
	); err != nil {
		c.cfg.L.Error("failed to update the followed video channels", zap.Error(err))
	}
}

func (c *Cache) add(fr framer.Frame) {
	for key, s := range fr.Entries() {
		if s.DataType != telem.VideoT {
			continue
		}
		c.mu.Lock()
		if e, ok := c.mu.entries[key]; ok {
			e.add(s, c.cfg)
		}
		c.mu.Unlock()
	}
}

// Bridge returns the cached frames of the channel that come before alignment, from
// the channel's last keyframe. A viewer whose first frame of the channel is at
// alignment decodes from the first returned frame on. Bridge returns false when the
// cache holds no keyframe that connects to alignment.
func (c *Cache) Bridge(
	key channel.Key,
	alignment telem.Alignment,
) ([]telem.Series, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.mu.entries[key]
	if !ok || len(e.series) == 0 {
		return nil, false
	}
	var bridge []telem.Series
	for _, s := range e.series {
		bounds := s.AlignmentBounds()
		switch {
		case alignment < bounds.Lower:
			return nil, false
		case alignment == bounds.Lower:
			return bridge, true
		case alignment < bounds.Upper:
			n := (alignment.SampleIndex() - bounds.Lower.SampleIndex()) / s.Multiple()
			return append(bridge, s.Slice(0, int(n))), true
		}
		bridge = append(bridge, s)
	}
	return bridge, alignment == e.end
}

// Close stops the Cache. It is not safe to call Close more than once.
func (c *Cache) Close() error {
	if c.disconnect != nil {
		c.disconnect()
	}
	c.requests.Close()
	return c.shutdown.Close()
}
