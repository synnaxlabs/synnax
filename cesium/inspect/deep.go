// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package inspect

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

const (
	// deepReadBufferSize buffers deep scans so a single read covers thousands of
	// samples.
	deepReadBufferSize = 64 * 1024
	// timestampSize is the byte size of one index-channel sample.
	timestampSize = 8
	// varLenPrefixSize is the byte size of a variable-length sample's length prefix.
	varLenPrefixSize = 4
)

// inspectDeep runs the checks that read sample bytes for one channel, filling in the
// report's sample counts where the structural pass could not. The error return is
// reserved for a canceled context; unreadable files were already reported by the
// structural pass and are skipped here.
func inspectDeep(
	ctx context.Context,
	cfg Config,
	chFS fs.FS,
	ch channel.Channel,
	records []domain.Record,
	rep *ChannelReport,
) error {
	files := &fileSet{fs: chFS, open: make(map[uint16]fs.File)}
	defer func() {
		if err := files.close(); err != nil {
			cfg.L.Warn("closing data files after deep inspection", zap.Error(err))
		}
	}()
	if ch.IsIndex {
		return inspectIndexContent(ctx, rep, files, records)
	}
	if ch.DataType.IsVariable() {
		return inspectVarLenWalk(ctx, rep, files, records)
	}
	return nil
}

// inspectIndexContent verifies that an index channel's stored timestamps agree with
// its domain bounds: samples increase strictly and stay within [Start, End).
func inspectIndexContent(
	ctx context.Context,
	rep *ChannelReport,
	files *fileSet,
	records []domain.Record,
) error {
	var v violation
	for i, r := range records {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Partial trailing samples are already reported by CheckDensityAlign.
		count := int64(r.Size) / timestampSize
		if count == 0 {
			continue
		}
		reader, ok := files.get(r.FileKey, r)
		if !ok {
			continue
		}
		var (
			buf  [timestampSize]byte
			prev telem.TimeStamp
		)
		for j := range count {
			if _, err := io.ReadFull(reader, buf[:]); err != nil {
				// Truncated files are already reported by CheckFileBounds.
				break
			}
			ts := telem.TimeStamp(telem.ByteOrder.Uint64(buf[:]))
			if ts.Before(r.Start) || ts.AfterEq(r.End) {
				v.note(fmt.Sprintf(
					"domain %d %v: sample %d (%v) outside bounds",
					i, r.TimeRange, j, ts,
				))
				break
			}
			if j > 0 && ts.BeforeEq(prev) {
				v.note(fmt.Sprintf(
					"domain %d %v: sample %d (%v) does not increase",
					i, r.TimeRange, j, ts,
				))
				break
			}
			prev = ts
		}
	}
	if v.count > 0 {
		rep.Findings = append(rep.Findings, newFinding(
			CheckIndexContent, rep.Key, v.first,
			"%d of %d domains hold timestamps that disagree with their bounds",
			v.count, len(records),
		))
	}
	return nil
}

// inspectVarLenWalk walks every variable-length domain's length prefixes, verifying
// that they land exactly on the domain's end, and counts samples along the way.
func inspectVarLenWalk(
	ctx context.Context,
	rep *ChannelReport,
	files *fileSet,
	records []domain.Record,
) error {
	var (
		v       violation
		samples int64
		exact   = true
	)
	for i, r := range records {
		if err := ctx.Err(); err != nil {
			return err
		}
		reader, ok := files.get(r.FileKey, r)
		if !ok {
			exact = false
			continue
		}
		var (
			buf    [varLenPrefixSize]byte
			offset = uint64(0)
			size   = uint64(r.Size)
		)
		for offset < size {
			if offset+varLenPrefixSize > size {
				v.note(fmt.Sprintf(
					"domain %d %v: %d trailing bytes after the last sample",
					i, r.TimeRange, size-offset,
				))
				exact = false
				break
			}
			if _, err := io.ReadFull(reader, buf[:]); err != nil {
				// Truncated files are already reported by CheckFileBounds.
				exact = false
				break
			}
			length := uint64(telem.ByteOrder.Uint32(buf[:]))
			if offset+varLenPrefixSize+length > size {
				v.note(fmt.Sprintf(
					"domain %d %v: sample at offset %d extends %d bytes past the "+
						"domain's end",
					i, r.TimeRange, offset,
					offset+varLenPrefixSize+length-size,
				))
				exact = false
				break
			}
			if _, err := reader.Discard(int(length)); err != nil {
				exact = false
				break
			}
			offset += varLenPrefixSize + length
			samples++
		}
	}
	if v.count > 0 {
		rep.Findings = append(rep.Findings, newFinding(
			CheckVarLenWalk, rep.Key, v.first,
			"%d of %d domains hold corrupt sample framing",
			v.count, len(records),
		))
	}
	rep.Stats.Samples = samples
	rep.Stats.SamplesExact = exact
	return nil
}

// fileSet lazily opens data files read-only and closes them together. Files that fail
// to open are remembered so each is attempted once.
type fileSet struct {
	fs     fs.FS
	open   map[uint16]fs.File
	failed set.Set[uint16]
}

// get returns a buffered reader over the record's byte region, or ok false when the
// backing file cannot be opened.
func (s *fileSet) get(key uint16, r domain.Record) (*bufio.Reader, bool) {
	f, exists := s.open[key]
	if !exists {
		if s.failed.Contains(key) {
			return nil, false
		}
		var err error
		if f, err = openRO(s.fs, domain.DataFileName(key)); err != nil {
			if s.failed == nil {
				s.failed = set.New[uint16]()
			}
			s.failed.Add(key)
			return nil, false
		}
		s.open[key] = f
	}
	section := io.NewSectionReader(f, int64(r.Offset), int64(r.Size))
	return bufio.NewReaderSize(section, deepReadBufferSize), true
}

// close closes every opened data file.
func (s *fileSet) close() (err error) {
	for _, f := range s.open {
		err = errors.Combine(err, f.Close())
	}
	return err
}
