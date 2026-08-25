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
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/migrate"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

const (
	// metaFileName mirrors the engine's meta file name.
	metaFileName = "meta.json"
	// metaTempFileName mirrors the engine's transient meta file name.
	metaTempFileName = "meta.json.tmp"
	// gcArtifactSuffix and tempArtifactSuffix mark files left by an interrupted GC.
	gcArtifactSuffix   = "_gc"
	tempArtifactSuffix = "_temp"
	// garbageWarnRatio is the garbage ratio at which CheckGarbage fires.
	garbageWarnRatio = 0.5
	// garbageWarnFloor is the minimum garbage size for CheckGarbage to fire, so tiny
	// databases with a few stale bytes stay quiet.
	garbageWarnFloor = telem.Megabyte
	// tinyWarnCount and tinyWarnRatio gate CheckTinyDomain: sparse channels
	// legitimately hold small domains, so the check fires only when tiny domains both
	// are numerous and dominate the channel.
	tinyWarnCount = 8
	tinyWarnRatio = 0.5
)

// violation aggregates repeated occurrences of one defect so findings stay bounded:
// one finding per check per channel, carrying the count and the first example.
type violation struct {
	count int
	first string
}

// note records one occurrence with the given subject.
func (v *violation) note(subject string) {
	if v.count == 0 {
		v.first = subject
	}
	v.count++
}

// inspectChannel inspects a single channel directory. Problems with the data become
// findings on the returned report; the error return is reserved for a canceled
// context.
func inspectChannel(
	ctx context.Context,
	cfg Config,
	key channel.Key,
	chFS fs.FS,
	metas map[channel.Key]metaResult,
) (ChannelReport, error) {
	rep := ChannelReport{Key: key}
	addf := func(check Check, subject, format string, args ...any) {
		rep.Findings = append(
			rep.Findings, newFinding(check, key, subject, format, args...),
		)
	}

	m := metas[key]
	metaOK := m.err == nil
	if metaOK {
		rep.Channel = m.channel
		if err := m.channel.Validate(); err != nil {
			addf(CheckMeta, metaFileName, "invalid channel metadata: %v", err)
		}
		state := migrate.Migrate(migrate.DBState{Channel: m.channel})
		if state.ShouldIgnoreChannel {
			addf(
				CheckIgnored, metaFileName,
				"the engine silently ignores this channel on open",
			)
		}
	} else {
		addf(CheckMeta, metaFileName, "%v", m.err)
	}
	inspectIndexRef(key, m, metaOK, metas, addf)

	entries, err := chFS.List("")
	if err != nil {
		addf(CheckMeta, "", "channel directory could not be listed: %v", err)
		return rep, nil
	}

	var (
		dataFiles = make(map[uint16]telem.Size)
		diskBytes telem.Size
	)
	for _, entry := range entries {
		name := entry.Name()
		diskBytes += telem.Size(entry.Size())
		switch {
		case entry.IsDir():
			addf(CheckOrphanFile, name, "unexpected directory in channel directory")
		case name == metaFileName,
			name == domain.IndexFileName,
			name == domain.CounterFileName:
		case name == metaTempFileName,
			strings.HasSuffix(name, gcArtifactSuffix),
			strings.HasSuffix(name, tempArtifactSuffix):
			addf(CheckArtifacts, name, "leftover from an interrupted GC or delete")
		case strings.HasSuffix(name, domain.Extension):
			fk, err := strconv.Atoi(strings.TrimSuffix(name, domain.Extension))
			if err != nil || fk <= 0 || fk > int(^uint16(0)) {
				addf(CheckOrphanFile, name, "unrecognized file in channel directory")
				continue
			}
			dataFiles[uint16(fk)] = telem.Size(entry.Size())
		default:
			addf(CheckOrphanFile, name, "unrecognized file in channel directory")
		}
	}
	rep.Stats.DiskBytes = diskBytes

	if metaOK && m.channel.Virtual {
		// A virtual channel stores nothing beyond its metadata.
		for fk := range dataFiles {
			addf(
				CheckOrphanFile, domain.DataFileName(fk),
				"virtual channels store no data",
			)
		}
		return rep, nil
	}

	counter, counterKnown := readCounter(cfg, chFS)
	if !counterKnown && len(dataFiles) > 0 {
		addf(
			CheckIndexDecode, domain.CounterFileName,
			"counter file missing or unreadable; file-key bound unknown",
		)
	}

	records, decoded := readIndex(chFS, len(dataFiles), addf)
	stats := inspectRecords(cfg, records, dataFiles, counter, counterKnown, m, addf)
	stats.DiskBytes = diskBytes
	rep.Stats = stats

	if *cfg.Deep && metaOK && decoded {
		if err := inspectDeep(ctx, cfg, chFS, m.channel, records, &rep); err != nil {
			return ChannelReport{}, err
		}
	}
	return rep, nil
}

// inspectIndexRef verifies that a data channel's index channel exists and is an
// index.
func inspectIndexRef(
	key channel.Key,
	m metaResult,
	metaOK bool,
	metas map[channel.Key]metaResult,
	addf func(check Check, subject, format string, args ...any),
) {
	if !metaOK || m.channel.Virtual || m.channel.IsIndex ||
		m.channel.Index == 0 || m.channel.Index == key {
		return
	}
	idx, exists := metas[m.channel.Index]
	switch {
	case !exists:
		addf(
			CheckIndexRef, "",
			"index channel %d has no directory", m.channel.Index,
		)
	case idx.err != nil:
		addf(
			CheckIndexRef, "",
			"index channel %d metadata is unreadable: %v", m.channel.Index, idx.err,
		)
	case !idx.channel.IsIndex:
		addf(
			CheckIndexRef, "",
			"channel %d is not an index channel", m.channel.Index,
		)
	}
}

// readCounter reads the file-key counter. An absent or empty counter file reads as
// zero, matching the engine; known is false when the file cannot be read at all.
func readCounter(cfg Config, chFS fs.FS) (value int32, known bool) {
	f, err := openRO(chFS, domain.CounterFileName)
	if err != nil {
		return 0, false
	}
	defer func() {
		if err := f.Close(); err != nil {
			cfg.L.Warn("closing counter file", zap.Error(err))
		}
	}()
	var buf [4]byte
	if _, err := f.ReadAt(buf[:], 0); err != nil {
		// An empty counter file reads as zero, matching the engine's counter.
		return 0, true
	}
	return int32(telem.ByteOrder.Uint32(buf[:])), true
}

// readIndex reads and decodes the channel's index file. decoded is false when the
// file could not be read at all.
func readIndex(
	chFS fs.FS,
	dataFileCount int,
	addf func(check Check, subject, format string, args ...any),
) (records []domain.Record, decoded bool) {
	b, err := readAll(chFS, domain.IndexFileName)
	if err != nil {
		if dataFileCount > 0 {
			addf(
				CheckIndexDecode, domain.IndexFileName,
				"index file unreadable with %d data files present: %v",
				dataFileCount, err,
			)
		}
		return nil, false
	}
	if len(b)%domain.RecordSize != 0 {
		addf(
			CheckIndexDecode, domain.IndexFileName,
			"index size %d is not a multiple of %d; trailing partial record dropped",
			len(b), domain.RecordSize,
		)
	}
	return domain.DecodeRecords(b), true
}

// inspectRecords runs every structural check over the decoded index and computes the
// channel's statistics.
func inspectRecords(
	cfg Config,
	records []domain.Record,
	dataFiles map[uint16]telem.Size,
	counter int32,
	counterKnown bool,
	m metaResult,
	addf func(check Check, subject, format string, args ...any),
) ChannelStats {
	var (
		stats                                 ChannelStats
		order, overlap, bounds, fkey, fbounds violation
		dens, farPast, nearEpoch, farFuture   violation
		micro                                 violation
		missing                               = make(map[uint16]int)
		referenced                            = make(map[uint16]telem.Size)
		sizes                                 = make([]telem.Size, 0, len(records))
		spans                                 = make([]telem.TimeSpan, 0, len(records))
		density                               telem.Density
		samples                               int64
	)
	if m.err == nil && !m.channel.DataType.IsVariable() {
		density = m.channel.DataType.Density()
	}
	now := cfg.Now()
	for i, r := range records {
		subject := fmt.Sprintf("domain %d %v", i, r.TimeRange)
		size := telem.Size(r.Size)
		if i > 0 {
			prev := records[i-1]
			if r.Start.Before(prev.Start) {
				order.note(subject)
			}
			gap := prev.End.Span(r.Start)
			if gap < 0 {
				overlap.note(subject)
			} else if gap > 0 {
				stats.Gaps++
				if gap <= cfg.MicroGapMax {
					stats.MicroGaps++
					micro.note(subject)
				}
			}
		}
		if r.Span() <= 0 {
			bounds.note(subject)
		}
		if r.FileKey == 0 || (counterKnown && int32(r.FileKey) > counter) {
			fkey.note(subject)
		}
		if fileSize, exists := dataFiles[r.FileKey]; exists {
			if telem.Size(r.Offset)+size > fileSize {
				fbounds.note(subject)
			}
		} else {
			missing[r.FileKey]++
		}
		referenced[r.FileKey] += size
		if density != 0 {
			if r.Size%uint32(density) != 0 {
				dens.note(subject)
			}
			samples += density.SampleCount(size)
		}
		if r.Start >= 0 && r.Start.BeforeEq(cfg.NearEpochMax) {
			nearEpoch.note(subject)
		} else if r.Start.Before(cfg.FarPast) {
			farPast.note(subject)
		}
		if r.End.After(now.Add(cfg.FarFutureSlack)) {
			farFuture.note(subject)
		}
		if size < cfg.TinyDomainSize || r.Span() < cfg.TinyDomainSpan {
			stats.TinyDomains++
		}
		stats.LiveBytes += size
		sizes = append(sizes, size)
		spans = append(spans, r.Span())
	}

	emit := func(v violation, check Check, what string) {
		if v.count > 0 {
			addf(check, v.first, "%d of %d domains %s", v.count, len(records), what)
		}
	}
	emit(order, CheckDomainOrder, "start before their predecessor")
	emit(overlap, CheckDomainOverlap, "overlap their predecessor")
	emit(bounds, CheckDomainBounds, "have an empty or inverted time range")
	emit(fkey, CheckFileKey, "reference file key zero or above the counter")
	emit(fbounds, CheckFileBounds, "extend past the end of their data file")
	emit(dens, CheckDensityAlign, "hold a partial trailing sample")
	emit(nearEpoch, CheckTimeBounds, fmt.Sprintf(
		"start within %v of the Unix epoch (elapsed-time-as-timestamp signature)",
		telem.TimeSpan(cfg.NearEpochMax),
	))
	emit(farPast, CheckTimeBounds, fmt.Sprintf("start before %v", cfg.FarPast))
	emit(farFuture, CheckTimeBounds, fmt.Sprintf(
		"end more than %v after the current time", cfg.FarFutureSlack,
	))
	if micro.count > 0 {
		addf(
			CheckMicroGap, micro.first,
			"%d gaps of at most %v between adjacent domains",
			micro.count, cfg.MicroGapMax,
		)
	}
	for fk, count := range missing {
		addf(
			CheckMissingFile, domain.DataFileName(fk),
			"data file referenced by %d domains does not exist", count,
		)
	}

	// The engine stores 80% of the configured file size as its cap and compares
	// per-file garbage against GCThreshold times that stored value; mirror it.
	gcEligibleFloor := telem.Size(
		float64(cfg.GCThreshold) * 0.8 * float64(cfg.FileSize),
	)
	var dataBytes telem.Size
	for fk, fileSize := range dataFiles {
		dataBytes += fileSize
		ref := referenced[fk]
		if ref == 0 && fileSize > 0 {
			addf(
				CheckOrphanFile, domain.DataFileName(fk),
				"data file (%v) is referenced by no domain", fileSize,
			)
		}
		if fileSize > ref {
			garbage := fileSize - ref
			stats.GarbageBytes += garbage
			if gcEligibleFloor > 0 && garbage >= gcEligibleFloor {
				stats.GCEligibleFiles++
			}
		}
	}
	if dataBytes > 0 {
		stats.GarbageRatio = float64(stats.GarbageBytes) / float64(dataBytes)
	}
	if stats.GarbageRatio >= garbageWarnRatio &&
		stats.GarbageBytes >= garbageWarnFloor {
		addf(
			CheckGarbage, "",
			"garbage is %.0f%% of %v of data-file bytes",
			stats.GarbageRatio*100, dataBytes,
		)
	}
	if stats.TinyDomains >= tinyWarnCount &&
		float64(stats.TinyDomains) >= tinyWarnRatio*float64(len(records)) {
		addf(
			CheckTinyDomain, "",
			"%d of %d domains are below %v or %v",
			stats.TinyDomains, len(records), cfg.TinyDomainSize, cfg.TinyDomainSpan,
		)
	}

	stats.Domains = len(records)
	stats.Files = len(dataFiles)
	stats.DomainSizes = newDistribution(sizes)
	stats.DomainSpans = newDistribution(spans)
	if len(records) > 0 {
		stats.TimeRange = telem.TimeRange{
			Start: records[0].Start,
			End:   records[len(records)-1].End,
		}
	}
	if density != 0 {
		stats.Samples = samples
		stats.SamplesExact = true
	}
	return stats
}

// openRO opens the named file read-only, retrying once to tolerate a live engine
// replacing files mid-scan.
func openRO(dataFS fs.FS, name string) (fs.File, error) {
	f, err := dataFS.Open(name, os.O_RDONLY)
	if err == nil {
		return f, nil
	}
	return dataFS.Open(name, os.O_RDONLY)
}

// readAll reads the named file fully, retrying once to tolerate a live engine
// truncating or replacing it mid-read.
func readAll(dataFS fs.FS, name string) (b []byte, err error) {
	for range 2 {
		if b, err = tryReadAll(dataFS, name); err == nil {
			return b, nil
		}
	}
	return nil, err
}

func tryReadAll(dataFS fs.FS, name string) (b []byte, err error) {
	f, err := dataFS.Open(name, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	b = make([]byte, info.Size())
	if len(b) == 0 {
		return b, nil
	}
	if _, err = f.ReadAt(b, 0); err != nil {
		return nil, err
	}
	return b, nil
}
