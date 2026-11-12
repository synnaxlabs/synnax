// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// TruncateToIndex truncates each domain to the samples that its index has timestamps
// for, and logs each truncation. A crash between the saves of a channel and its index
// leaves such domains. Before each truncation, it copies the cut bytes to a .truncated
// file in the channel directory that the log names. It reuses a file that already holds
// those bytes, and never overwrites any other file. When the copy fails, it returns the
// error and does not truncate.
// TruncateToIndex must run before any writer or iterator opens on the DB. It does
// nothing for an index channel.
func (db *DB) TruncateToIndex(ctx context.Context) error {
	if db.cfg.Channel.IsIndex {
		return nil
	}
	cuts, err := db.findCuts(ctx)
	if err != nil {
		return db.wrapError(err)
	}
	for _, c := range cuts {
		cutStart := func(
			context.Context,
			telem.TimeStamp,
			telem.TimeStamp,
		) (telem.Size, telem.TimeStamp, error) {
			return c.offset, c.start, nil
		}
		// Delete calls the end resolver only for a domain that starts where the cut
		// ends. That domain stays whole.
		keepEnd := func(
			_ context.Context,
			_ telem.TimeStamp,
			ts telem.TimeStamp,
		) (telem.Size, telem.TimeStamp, error) {
			return 0, ts, nil
		}
		var file string
		if file, err = db.quarantine(ctx, c); err == nil {
			err = db.domain.Delete(ctx, c.start.Range(c.domain.End), cutStart, keepEnd)
		}
		if err != nil {
			return db.wrapError(errors.Wrapf(err, "domain %v", c.domain))
		}
		db.cfg.L.Warn(
			"truncated a domain to the samples its index has timestamps for",
			zap.Stringer("channel", db.cfg.Channel),
			zap.Stringer("domain", c.domain),
			zap.Int64("dropped_samples", c.dropped),
			zap.String("quarantine", file),
		)
	}
	db.resolver.invalidate()
	return nil
}

// quarantine copies the bytes of c to the first free name of <start>-<end>.truncated,
// <start>-<end>.1.truncated, and so on, and returns the name. It reuses a file that
// already holds those bytes, so a retried truncation does not add a copy.
func (db *DB) quarantine(ctx context.Context, c cut) (string, error) {
	base := fmt.Sprintf("%d-%d", c.start, c.domain.End)
	name := base + ".truncated"
	for n := 1; ; n++ {
		info, err := db.cfg.FS.Stat(name)
		if os.IsNotExist(err) {
			if err = db.copyCut(ctx, c, name); err != nil {
				return "", err
			}
			return name, nil
		}
		if err != nil {
			return "", err
		}
		if info.Size() == int64(c.size) {
			held, err := db.holdsCut(ctx, c, name)
			if err != nil {
				return "", err
			}
			if held {
				return name, nil
			}
		}
		name = fmt.Sprintf("%s.%d.truncated", base, n)
	}
}

func (db *DB) readCut(ctx context.Context, c cut, f func(io.Reader) error) (err error) {
	iter := db.domain.OpenIterator(domain.IterRange(c.domain))
	defer func() { err = errors.Combine(err, iter.Close()) }()
	if !iter.SeekFirst(ctx) || iter.TimeRange() != c.domain {
		return errors.Newf("domain %v not found", c.domain)
	}
	r, err := iter.OpenReader(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, r.Close()) }()
	return f(io.NewSectionReader(r, int64(c.offset), int64(c.size)))
}

func (db *DB) holdsCut(ctx context.Context, c cut, name string) (held bool, err error) {
	f, err := db.cfg.FS.Open(name, os.O_RDONLY)
	if err != nil {
		return false, err
	}
	defer func() {
		if err = errors.Combine(err, f.Close()); err != nil {
			held = false
		}
	}()
	err = db.readCut(ctx, c, func(cutBytes io.Reader) error {
		var (
			a = make([]byte, 32*1024)
			b = make([]byte, len(a))
		)
		for {
			n, err := io.ReadFull(cutBytes, a)
			if err != nil &&
				!errors.Is(err, io.ErrUnexpectedEOF) &&
				!errors.Is(err, io.EOF) {
				return err
			}
			if n == 0 {
				held = true
				return nil
			}
			if _, err = io.ReadFull(f, b[:n]); err != nil {
				return err
			}
			if !bytes.Equal(a[:n], b[:n]) {
				return nil
			}
		}
	})
	if err != nil {
		return false, err
	}
	return held, nil
}

func (db *DB) copyCut(ctx context.Context, c cut, name string) (err error) {
	tmp := name + ".tmp"
	f, err := db.cfg.FS.Open(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return err
	}
	err = db.readCut(ctx, c, func(r io.Reader) error {
		if _, err := io.Copy(f, r); err != nil {
			return err
		}
		return f.Sync()
	})
	if err = errors.Combine(err, f.Close()); err == nil {
		err = db.cfg.FS.Rename(tmp, name)
	}
	if err != nil {
		return errors.Combine(err, db.cfg.FS.Remove(tmp))
	}
	return nil
}

// cut is the tail of a domain that its index has no timestamps for.
type cut struct {
	domain  telem.TimeRange
	start   telem.TimeStamp
	offset  telem.Size
	size    telem.Size
	dropped int64
}

func (db *DB) findCuts(ctx context.Context) (cuts []cut, err error) {
	iter := db.domain.OpenIterator(domain.IterRange(telem.TimeRangeMax))
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for ok := iter.SeekFirst(ctx); ok; ok = iter.Next() {
		var (
			c     cut
			found bool
		)
		if c, found, err = db.findCut(ctx, iter); err != nil {
			return nil, errors.Wrapf(err, "domain %v", iter.TimeRange())
		}
		if found {
			cuts = append(cuts, c)
		}
	}
	return cuts, nil
}

func (db *DB) findCut(ctx context.Context, iter *domain.Iterator) (cut, bool, error) {
	tr := iter.TimeRange()
	end, covered, err := db.index().Coverage(ctx, tr)
	if err != nil {
		return cut{}, false, err
	}
	// A domain ends 1 ns after its last sample, so a domain that ends inside the run of
	// its index has a timestamp for every sample.
	if tr.End <= end {
		return cut{}, false, nil
	}
	count, err := db.resolver.domainSampleCount(ctx, iter)
	if err != nil || count <= covered {
		return cut{}, false, err
	}
	offset, err := db.resolver.byteOffset(ctx, iter, covered)
	if err != nil {
		return cut{}, false, err
	}
	return cut{
		domain:  tr,
		start:   max(end, tr.Start),
		offset:  offset,
		size:    iter.Size() - offset,
		dropped: count - covered,
	}, true, nil
}
