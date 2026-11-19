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
// deletes s<k>, so a crash can land in each step of the delete. Each ack is Visible,
// because Cesium syncs no write.
type crashEngine struct {
	fs      xfs.FS
	ts      xfs.FS
	pebble  *pebble.DB
	db      *gorp.DB
	cesium  *cesium.DB
	svc     *channel.Service
	closers []func() error
	clock   channel.Channel
	// broken is the first invariant that Recovered found broken, for Audit.
	broken error
}

func openCrashEngine(ctx context.Context, fs xfs.FS) (_ *crashEngine, err error) {
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
	logger := pebblekv.NewNoopLogger()
	opts := &pebble.Options{
		FS:                 pebblefs.New(kv),
		FormatMajorVersion: pebble.FormatNewest,
		Logger:             logger,
	}
	// An open that fails leaves the scheduler of Pebble registered, and its granter
	// running. The workload runs no compaction, so a scheduler with no granter is
	// enough.
	scheduler := pebble.NewConcurrencyLimitSchedulerWithNoPeriodicGrantingForTest()
	opts.Experimental.CompactionScheduler = scheduler
	if e.pebble, err = pebble.Open("", opts); err != nil {
		return nil, err
	}
	// Closing Gorp closes the store and Pebble under it.
	store := pebblekv.Wrap(e.pebble, logger)
	e.db = gorp.Wrap(store)
	e.closers = append(e.closers, e.db.Close, GoRun(ctx, e.db.Run))
	if e.cesium, err = cesium.Open(ctx, "", cesium.WithFS(e.ts)); err != nil {
		return nil, err
	}
	e.closers = append(e.closers, e.cesium.Close, GoRun(ctx, e.cesium.Run))
	if err = e.openServices(ctx, store); err != nil {
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
	if err := e.db.WithTx(ctx, func(tx gorp.Tx) error {
		return e.svc.NewWriter(tx).Delete(ctx, scratch.Key(), false)
	}); err != nil {
		return 0, err
	}
	return crash.Visible, e.removed()
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
			kept, err := e.read(ctx, scratch)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(kept, data) && e.broken == nil {
				e.broken = errors.Newf(
					"%s kept its record and lost its samples",
					scratch.Name,
				)
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

func (e *crashEngine) Audit(context.Context) error { return e.broken }

func (e *crashEngine) Close() error {
	var err error
	for _, c := range slices.Backward(e.closers) {
		err = errors.Combine(err, c())
	}
	return err
}

// runOnCore runs the harness on the adapter with one fault profile and one run key.
func runOnCore(ctx context.Context, profile crash.Profile, runKey uint64) error {
	_, err := crash.Run(ctx, crash.Config{
		RunKey: runKey,
		Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
			e, err := openCrashEngine(ctx, fs)
			if err != nil {
				return nil, err
			}
			return e, nil
		},
		Commits: 4,
		Stride:  1,
		Profile: profile,
	})
	return err
}

// Each entry is what the harness finds in the channel services today, and why. A delete
// removes the Cesium data before the record delete commits, and syncs neither the
// removal nor the create of a directory. So a crash keeps the Cesium channel of a
// deleted record (R5), and Partial loses a meta file first (R4).
var _ = DescribeTable("Crash harness on the channel services",
	func(
		ctx SpecContext,
		profile crash.Profile,
		runKey uint64,
		rule crash.Rule,
		cause string,
	) {
		var v *crash.Violation
		err := runOnCore(ctx, profile, runKey)
		Expect(errors.As(err, &v)).To(BeTrue(), "no violation: %v", err)
		Expect(v.Rule).To(Equal(rule))
		Expect(v.Detail).To(ContainSubstring(cause))
	},
	Entry("Strict: R5", crash.Strict, uint64(0), crash.R5,
		"cesium channel 1048579 has no record"),
	Entry("Partial: R4", crash.Partial, uint64(1), crash.R4, "data_type: required"),
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
			Expect(e.cesium.DeleteChannel(scratch.Key().StorageKey())).To(Succeed())
			Expect(e.removed()).To(Succeed())
			MustSucceed(e.Recovered(ctx))
			Expect(
				e.Audit(ctx),
			).To(MatchError("s1 kept its record and lost its samples"))
		},
	)
})
