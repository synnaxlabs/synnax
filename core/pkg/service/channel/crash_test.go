// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/pebble/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/io/fs/testutil/pebblefs"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// crashEngine runs the channel services of the Core on one disk: Gorp on Pebble in kv,
// and Cesium in cesium. Record k is the channel d<k>, with the payload as its samples.
// Each commit also writes the payload to the channel s<k>, writes the disk back, and
// deletes s<k>, so a crash can land in each step of the delete. Each ack is Durable:
// the commit writes the disk back before it deletes s<k>.
type crashEngine struct {
	fs      xfs.FS
	ts      xfs.FS
	pebble  *pebble.DB
	db      *gorp.DB
	store   kv.DB
	cesium  *cesium.DB
	svc     *channel.Service
	closers []func() error
	clock   channel.Channel
	// broken is the first invariant that Recovered found broken, for Audit.
	broken error
}

// durableDirs syncs the directory of each create, remove, and rename of Cesium, as a
// Cesium that makes its directory changes durable would. It syncs no file.
type durableDirs struct{ xfs.FS }

// parent returns the directory of name, "" for the root of the FS.
func parent(name string) string {
	if dir := path.Dir(name); dir != "." {
		return dir
	}
	return ""
}

func (d durableDirs) Open(name string, flag int) (xfs.File, error) {
	f, err := d.FS.Open(name, flag)
	if err != nil || flag&os.O_CREATE == 0 {
		return f, err
	}
	return f, d.SyncDir(parent(name))
}

func (d durableDirs) Sub(name string) (xfs.FS, error) {
	sub, err := d.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return durableDirs{FS: sub}, d.SyncDir(parent(name))
}

func (d durableDirs) Remove(name string) error {
	if err := d.FS.Remove(name); err != nil {
		return err
	}
	return d.SyncDir(parent(name))
}

func (d durableDirs) Rename(from, to string) error {
	if err := d.FS.Rename(from, to); err != nil {
		return err
	}
	return errors.Join(d.SyncDir(parent(from)), d.SyncDir(parent(to)))
}

func openCrashEngine(ctx context.Context, fs xfs.FS) (_ *crashEngine, err error) {
	return openEngine(ctx, fs, false)
}

// openEngine opens the engine on fs. With dirsDurable, Cesium makes each of its
// directory changes durable.
func openEngine(
	ctx context.Context,
	fs xfs.FS,
	dirsDurable bool,
) (_ *crashEngine, err error) {
	e := &crashEngine{fs: fs}
	defer func() {
		if err != nil {
			err = errors.Combine(err, e.Close())
		}
	}()
	kv, err := fs.Sub("kv")
	if err != nil {
		return nil, err
	}
	if e.ts, err = fs.Sub("cesium"); err != nil {
		return nil, err
	}
	if dirsDurable {
		if err = fs.SyncDir(""); err != nil {
			return nil, err
		}
		e.ts = durableDirs{FS: e.ts}
	}
	if err = e.openKV(ctx, kv); err != nil {
		return nil, err
	}
	if e.cesium, err = cesium.Open(ctx, "", cesium.WithFS(e.ts)); err != nil {
		return nil, err
	}
	e.closers = append(e.closers, e.cesium.Close, GoRun(ctx, e.cesium.Run))
	if err = e.openServices(ctx, e.store); err != nil {
		return nil, err
	}
	e.clock = channel.Channel{Name: "clock", DataType: telem.TimestampT, IsIndex: true}
	err = e.svc.NewRetrieve().
		Where(channel.MatchNames(e.clock.Name)).
		Entry(&e.clock).
		Exec(ctx, e.db)
	if errors.Is(err, query.ErrNotFound) {
		err = e.create(ctx, &e.clock)
	}
	return e, err
}

// openKV opens Pebble on fs, and Gorp on it.
func (e *crashEngine) openKV(ctx context.Context, fs xfs.FS) (err error) {
	logger := pebblekv.NewNoopLogger()
	opts := &pebble.Options{
		FS:                 pebblefs.New(fs),
		FormatMajorVersion: pebble.FormatNewest,
		Logger:             logger,
	}
	// An open that fails leaves the scheduler of Pebble registered, and its granter
	// running. The workload runs no compaction, so a scheduler with no granter is
	// enough.
	scheduler := pebble.NewConcurrencyLimitSchedulerWithNoPeriodicGrantingForTest()
	opts.Experimental.CompactionScheduler = scheduler
	if e.pebble, err = pebble.Open("", opts); err != nil {
		return err
	}
	// Closing Gorp closes the store and Pebble under it.
	e.store = pebblekv.Wrap(e.pebble, logger)
	e.db = gorp.Wrap(e.store)
	e.closers = append(e.closers, e.db.Close, GoRun(ctx, e.db.Run))
	return nil
}

// openServices opens the channel services of the Core on store and on the Gorp and
// Cesium of e.
func (e *crashEngine) openServices(ctx context.Context, store kv.DB) (err error) {
	dist, err := distchannel.NewService(ctx, distchannel.ServiceConfig{
		HostKey: 1,
		KV:      store,
		TS:      e.cesium,
	})
	if err != nil {
		return err
	}
	groups, err := group.OpenService(ctx, group.ServiceConfig{DB: e.db})
	if err != nil {
		return err
	}
	e.closers = append(e.closers, groups.Close)
	labels, err := label.OpenService(ctx, label.ServiceConfig{DB: e.db})
	if err != nil {
		return err
	}
	e.closers = append(e.closers, labels.Close)
	statuses, err := status.OpenService(ctx, status.ServiceConfig{DB: e.db})
	if err != nil {
		return err
	}
	e.closers = append(e.closers, statuses.Close)
	if e.svc, err = channel.OpenService(ctx, channel.ServiceConfig{
		Channel: dist,
		DB:      e.db,
		HostKey: 1,
		Status:  statuses,
	}); err != nil {
		return err
	}
	e.closers = append(e.closers, e.svc.Close)
	return nil
}

func (e *crashEngine) create(ctx context.Context, ch *channel.Channel) error {
	return e.db.WithTx(ctx, func(tx gorp.Tx) error {
		return e.svc.NewWriter(tx).Create(ctx, ch)
	})
}

// samples returns a channel named name for the samples of the clock.
func (e *crashEngine) samples(name string) channel.Channel {
	return channel.Channel{
		Name:       name,
		DataType:   telem.Uint8T,
		LocalIndex: e.clock.LocalKey,
	}
}

// stamps returns the timestamps of the record id with n payload bytes: one for each
// byte, from the second of id.
func stamps(id uint64, n int) telem.Series {
	start := telem.TimeStamp(id) * telem.SecondTS
	out := make([]telem.TimeStamp, n)
	for i := range out {
		out[i] = start + telem.TimeStamp(i)
	}
	return telem.NewSeries(out)
}

func (e *crashEngine) Commit(
	ctx context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	data := e.samples("d" + strconv.FormatUint(r.ID, 10))
	scratch := e.samples("s" + strconv.FormatUint(r.ID, 10))
	if err := e.create(ctx, &data); err != nil {
		return 0, err
	}
	if err := e.create(ctx, &scratch); err != nil {
		return 0, err
	}
	frame := telem.MultiFrame(
		[]cesium.ChannelKey{
			e.clock.Key().StorageKey(),
			data.Key().StorageKey(),
			scratch.Key().StorageKey(),
		},
		[]telem.Series{
			stamps(r.ID, len(r.Payload)),
			telem.NewSeries(r.Payload),
			telem.NewSeries(r.Payload),
		},
	)
	start := telem.TimeStamp(r.ID) * telem.SecondTS
	if err := e.cesium.Write(ctx, start, frame); err != nil {
		return 0, err
	}
	if err := crash.Writeback(e.fs); err != nil {
		return 0, err
	}
	// The samples of the scratch channel are durable from here on.
	if err := e.store.Set(ctx, written(scratch), nil); err != nil {
		return 0, err
	}
	if err := e.db.WithTx(ctx, func(tx gorp.Tx) error {
		return e.svc.NewWriter(tx).Delete(ctx, scratch.Key(), false)
	}); err != nil {
		return 0, err
	}
	return crash.Durable, e.removed()
}

// removed waits until Cesium removed the directory of each deleted channel, so the
// removal adds its events before the next commit.
func (e *crashEngine) removed() error {
	for range 1000 {
		infos, err := e.ts.List("")
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(infos, func(info xfs.FileInfo) bool {
			return strings.Contains(info.Name(), "-DELETE-")
		}) {
			return nil
		}
		time.Sleep(time.Millisecond)
	}
	return errors.New("cesium: a deleted channel was not removed")
}

// read returns the samples of the channel in Cesium, or nil when Cesium has no such
// channel.
func (e *crashEngine) read(ctx context.Context, ch channel.Channel) ([]byte, error) {
	frame, err := e.cesium.Read(ctx, telem.TimeRangeMax, ch.Key().StorageKey())
	if errors.Is(err, cesium.ErrChannelNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []byte
	for _, s := range frame.SeriesSlice() {
		out = append(out, s.Data...)
	}
	return out, nil
}

// Recovered returns a record for each channel d<k> with samples. It also checks the
// invariants of the delete order, for Audit: each channel directory in Cesium has a
// Gorp record, and a channel s<k> that still has its record holds the samples of d<k>.
func (e *crashEngine) Recovered(ctx context.Context) ([]crash.Record, error) {
	var channels []channel.Channel
	if err := e.svc.NewRetrieve().Entries(&channels).Exec(ctx, e.db); err != nil {
		return nil, err
	}
	stored := set.Set[cesium.ChannelKey]{}
	byName := map[string]channel.Channel{}
	for _, ch := range channels {
		stored.Add(ch.Key().StorageKey())
		byName[ch.Name] = ch
	}
	infos, err := e.ts.List("")
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		key, err := strconv.ParseUint(info.Name(), 10, 32)
		if err == nil && !stored.Contains(cesium.ChannelKey(key)) && e.broken == nil {
			e.broken = errors.Newf("cesium channel %d has no record", key)
		}
	}
	var records []crash.Record
	for _, ch := range channels {
		name, ok := strings.CutPrefix(ch.Name, "d")
		if !ok {
			continue
		}
		id, err := strconv.ParseUint(name, 10, 64)
		if err != nil {
			return nil, err
		}
		data, err := e.read(ctx, ch)
		if err != nil {
			return nil, err
		}
		if scratch, ok := byName["s"+name]; ok {
			if err = e.checkScratch(ctx, scratch, data); err != nil {
				return nil, err
			}
		}
		if len(data) > 0 {
			records = append(records, crash.Record{ID: id, Payload: data})
		}
	}
	slices.SortFunc(
		records,
		func(a, b crash.Record) int { return int(a.ID) - int(b.ID) },
	)
	return records, nil
}

// checkScratch notes, for Audit, a scratch channel that still has its record and lost
// the samples of its data channel, when they were durable before its delete began.
func (e *crashEngine) checkScratch(
	ctx context.Context,
	scratch channel.Channel,
	data []byte,
) error {
	durable, err := e.durable(scratch)
	if err != nil || !durable {
		return err
	}
	kept, err := e.read(ctx, scratch)
	if err != nil {
		return err
	}
	if !bytes.Equal(kept, data) && e.broken == nil {
		e.broken = errors.Newf("%s kept its record and lost its samples", scratch.Name)
	}
	return nil
}

// names returns the names of the files in dir.
func names(fs xfs.FS, dir string) []string {
	GinkgoHelper()
	infos := MustSucceed(fs.List(dir))
	out := make([]string, len(infos))
	for i, info := range infos {
		out[i] = info.Name()
	}
	return out
}

// written returns the key that marks the samples of ch as durable.
func written(ch channel.Channel) []byte { return []byte("crash/written/" + ch.Name) }

// durable reports if the samples of ch were durable before its delete began.
func (e *crashEngine) durable(ch channel.Channel) (bool, error) {
	_, closer, err := e.store.Get(written(ch))
	if errors.Is(err, query.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, closer.Close()
}

func (e *crashEngine) Audit(context.Context) error { return e.broken }

func (e *crashEngine) Close() error {
	var err error
	for _, c := range slices.Backward(e.closers) {
		err = errors.Combine(err, c())
	}
	return err
}

// runOnCore runs the harness under Strict on the adapter, and checks each crash image.
func runOnCore(ctx context.Context, dirsDurable bool) (crash.Result, error) {
	return crash.Run(ctx, crash.Config{
		Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
			e, err := openEngine(ctx, fs, dirsDurable)
			if err != nil {
				return nil, err
			}
			return e, nil
		},
		Commits:    4,
		Stride:     1,
		Profile:    crash.Strict,
		Exhaustive: true,
	})
}

// orphans returns the violation of each Cesium channel with the local key and no
// record.
func orphans(locals ...distchannel.LocalKey) []string {
	out := make([]string, len(locals))
	for i, local := range locals {
		key := distchannel.NewKey(1, local).StorageKey()
		out[i] = fmt.Sprintf(
			"R5: the audit failed: cesium channel %d has no record",
			key,
		)
	}
	return out
}

// emptyMeta returns the violation of an open that found the meta file of each channel
// with the local key empty, after cause.
func emptyMeta(cause string, locals ...distchannel.LocalKey) []string {
	out := make([]string, len(locals))
	for i, local := range locals {
		out[i] = fmt.Sprintf(
			"R4: %s: error decoding meta in folder for channel %d: failed to decode "+
				"channel.Channel: EOF",
			cause,
			distchannel.NewKey(1, local).StorageKey(),
		)
	}
	return out
}

// Each entry lists every violation that the harness finds in the channel services
// today, under Strict, where no draw decides what a crash keeps. Cesium syncs no
// removal and nothing sweeps, so a crash keeps the Cesium channel of each deleted
// record. When Cesium makes its directory changes durable and syncs no file, a delete
// that removes the Cesium data before its record delete is durable also takes the
// samples of a record that stays, and a crash stops the open on a channel directory
// with no meta file or with an empty one.
var _ = DescribeTable("Crash harness on the channel services",
	func(ctx SpecContext, dirsDurable bool, want []string) {
		result, err := runOnCore(ctx, dirsDurable)
		var first *crash.Violation
		Expect(errors.As(err, &first)).To(BeTrue(), "no violation: %v", err)
		Expect(result.Violations[0]).To(Equal(*first))
		found := set.Set[string]{}
		for _, v := range result.Violations {
			found.Add(string(v.Rule) + ": " + v.Detail)
		}
		Expect(found.Slice()).To(ConsistOf(want))
	},
	Entry("plain", false, orphans(3, 5, 7, 9)),
	Entry("with durable Cesium directories", true, slices.Concat(
		emptyMeta("the engine did not recover", 1, 4, 6, 8),
		emptyMeta("the engine did not recover after a crash in recovery", 1, 2),
		[]string{
			"R4: the engine did not recover: data_type: required\nname: required\n" +
				"index: non-indexed channel must have an index",
			"R5: the audit failed: s1 kept its record and lost its samples",
			"R5: the audit failed: s2 kept its record and lost its samples",
			"R5: the audit failed: s3 kept its record and lost its samples",
			"R5: the audit failed: s4 kept its record and lost its samples",
		},
	)),
)

var _ = Describe("Crash adapter on the channel services", func() {
	var (
		disk *fault.FS
		e    *crashEngine
	)
	BeforeEach(func(ctx SpecContext) {
		disk = MustSucceed(fault.New(fault.Config{}))
		e = MustSucceed(openCrashEngine(ctx, disk))
		DeferCleanup(func() { Expect(e.Close()).To(Succeed()) })
	})
	// write creates the channels d1 and s1, and writes payload to both.
	write := func(
		ctx context.Context,
		payload []byte,
	) (channel.Channel, channel.Channel) {
		GinkgoHelper()
		data, scratch := e.samples("d1"), e.samples("s1")
		Expect(e.create(ctx, &data)).To(Succeed())
		Expect(e.create(ctx, &scratch)).To(Succeed())
		Expect(e.cesium.Write(ctx, telem.SecondTS, telem.MultiFrame(
			[]cesium.ChannelKey{
				e.clock.Key().StorageKey(),
				data.Key().StorageKey(),
				scratch.Key().StorageKey(),
			},
			[]telem.Series{
				stamps(1, len(payload)),
				telem.NewSeries(payload),
				telem.NewSeries(payload),
			},
		))).To(Succeed())
		return data, scratch
	}

	It("Should recover the samples of each data channel after a reopen",
		func(ctx SpecContext) {
			write(ctx, []byte("taco"))
			Expect(e.Close()).To(Succeed())
			e = MustSucceed(openCrashEngine(ctx, disk))
			Expect(e.Recovered(ctx)).To(Equal([]crash.Record{
				{ID: 1, Payload: []byte("taco")},
			}))
			Expect(e.Audit(ctx)).To(Succeed())
		},
	)

	It("Should make each Cesium directory change durable at once, in that mode",
		func(ctx SpecContext) {
			Expect(e.Close()).To(Succeed())
			e = MustSucceed(openEngine(ctx, disk, true))
			Expect(e.cesium.CreateChannel(ctx, cesium.Channel{
				Key:      99,
				Name:     "stray",
				IsIndex:  true,
				DataType: telem.TimestampT,
			})).To(Succeed())
			Expect(e.cesium.Write(ctx, telem.SecondTS, telem.MultiFrame(
				[]cesium.ChannelKey{99},
				[]telem.Series{stamps(1, 1)},
			))).To(Succeed())
			image := MustSucceed(disk.Crash(fault.CrashConfig{}))
			Expect(names(image, "cesium/99")).To(Equal(names(disk, "cesium/99")))
			Expect(e.cesium.DeleteChannel(99)).To(Succeed())
			Expect(e.removed()).To(Succeed())
			image = MustSucceed(disk.Crash(fault.CrashConfig{}))
			Expect(MustSucceed(image.List("cesium"))).ToNot(ContainElement(
				WithTransform(os.FileInfo.Name, HavePrefix("99")),
			))
		},
	)

	It("Should report a Cesium channel with no record", func(ctx SpecContext) {
		Expect(e.cesium.CreateChannel(ctx, cesium.Channel{
			Key:      99,
			Name:     "stray",
			IsIndex:  true,
			DataType: telem.TimestampT,
		})).To(Succeed())
		MustSucceed(e.Recovered(ctx))
		Expect(e.Audit(ctx)).To(MatchError("cesium channel 99 has no record"))
	})

	It("Should report a deleted channel that kept its record and lost its samples",
		func(ctx SpecContext) {
			_, scratch := write(ctx, []byte("taco"))
			Expect(e.store.Set(ctx, written(scratch), nil)).To(Succeed())
			Expect(e.cesium.DeleteChannel(scratch.Key().StorageKey())).To(Succeed())
			Expect(e.removed()).To(Succeed())
			MustSucceed(e.Recovered(ctx))
			Expect(
				e.Audit(ctx),
			).To(MatchError("s1 kept its record and lost its samples"))
		},
	)
})
