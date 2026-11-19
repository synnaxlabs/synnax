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
	"context"
	"os"
	"slices"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// storageKey returns the Cesium key of the channel with the local key.
func storageKey(local distchannel.LocalKey) cesium.ChannelKey {
	return distchannel.NewKey(1, local).StorageKey()
}

// dataKey and scratchKey return the Cesium keys of d<k> and s<k>. The clock takes the
// local key 1, and each commit creates d<k> and then s<k>.
func dataKey(k int) cesium.ChannelKey {
	return storageKey(distchannel.LocalKey(2 * k))
}

func scratchKey(k int) cesium.ChannelKey {
	return storageKey(distchannel.LocalKey(2*k + 1))
}

// createPair creates the channels d<id> and s<id>, as Commit does.
func createPair(
	ctx context.Context,
	e *crashEngine,
	id uint64,
) (channel.Channel, channel.Channel) {
	GinkgoHelper()
	name := strconv.FormatUint(id, 10)
	data, scratch := e.samples("d"+name), e.samples("s"+name)
	Expect(e.create(ctx, &data)).To(Succeed())
	Expect(e.create(ctx, &scratch)).To(Succeed())
	return data, scratch
}

// writePair writes payload to the clock, data, and scratch, as Commit does.
func writePair(
	ctx context.Context,
	e *crashEngine,
	id uint64,
	data, scratch channel.Channel,
	payload []byte,
) error {
	return e.cesium.Write(ctx, telem.TimeStamp(id)*telem.SecondTS, telem.MultiFrame(
		[]cesium.ChannelKey{
			e.clock.Key().StorageKey(),
			data.Key().StorageKey(),
			scratch.Key().StorageKey(),
		},
		[]telem.Series{
			stamps(id, len(payload)),
			telem.NewSeries(payload),
			telem.NewSeries(payload),
		},
	))
}

// violations returns each violation of result as "rule: detail", once.
func violations(result crash.Result) []string {
	found := set.Set[string]{}
	for _, v := range result.Violations {
		found.Add(string(v.Rule) + ": " + v.Detail)
	}
	return found.Slice()
}

// liveLoss is the engine of the workload with a read that drops the first commit.
type liveLoss struct{ *crashEngine }

func (l liveLoss) Recovered(ctx context.Context) ([]crash.Record, error) {
	records, err := l.crashEngine.Recovered(ctx)
	if len(records) > 0 {
		records = records[1:]
	}
	return records, err
}

// recordFirst runs the delete of s<k> in the order that removes the record first: the
// record delete commits, then Cesium drops the channel.
type recordFirst struct{ *crashEngine }

func (e recordFirst) Commit(
	ctx context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	name := strconv.FormatUint(r.ID, 10)
	data, scratch := e.samples("d"+name), e.samples("s"+name)
	if err := e.create(ctx, &data); err != nil {
		return 0, err
	}
	if err := e.create(ctx, &scratch); err != nil {
		return 0, err
	}
	err := writePair(ctx, e.crashEngine, r.ID, data, scratch, r.Payload)
	if err != nil {
		return 0, err
	}
	if err := crash.Writeback(e.fs); err != nil {
		return 0, err
	}
	if err := e.store.Set(ctx, written(scratch), nil); err != nil {
		return 0, err
	}
	if err := e.db.WithTx(ctx, func(tx gorp.Tx) error {
		return gorp.NewDelete[channel.Key, channel.Channel]().
			Where(gorp.MatchKeys[channel.Key, channel.Channel](scratch.Key())).
			Exec(ctx, tx)
	}); err != nil {
		return 0, err
	}
	if err := e.cesium.DeleteChannel(scratch.Key().StorageKey()); err != nil {
		return 0, err
	}
	return crash.Visible, e.removed()
}

// loseNewest removes, once for each crash image and the crash in its recovery, the
// Cesium channel of the newest commit k whose samples are durable and whose s<k> Cesium
// removed. It reports if it removed one.
func loseNewest(image *fault.FS) bool {
	GinkgoHelper()
	if MustSucceed(image.Exists("lost")) {
		return false
	}
	f := MustSucceed(image.Open("lost", os.O_CREATE|os.O_WRONLY))
	Expect(f.Close()).To(Succeed())
	Expect(image.SyncDir("")).To(Succeed())
	for k := 4; k >= 1; k-- {
		d := "cesium/" + strconv.FormatUint(uint64(dataKey(k)), 10)
		s := "cesium/" + strconv.FormatUint(uint64(scratchKey(k)), 10)
		if !MustSucceed(image.Exists(d)) {
			continue
		}
		info, err := image.Stat(d + "/1.domain")
		if err != nil || info.Size() == 0 || MustSucceed(image.Exists(s)) {
			return false
		}
		Expect(image.Remove(d)).To(Succeed())
		return true
	}
	return false
}

var _ = Describe("Crash adapter check", func() {
	var disk *fault.FS
	BeforeEach(func() { disk = MustSucceed(fault.New(fault.Config{})) })

	Describe("durableDirs", func() {
		It("Should not make the bytes of a file durable with its rename",
			func(ctx SpecContext) {
				e := MustSucceed(openEngine(ctx, disk, true))
				DeferCleanup(func() { Expect(e.Close()).To(Succeed()) })
				Expect(e.cesium.CreateChannel(ctx, cesium.Channel{
					Key:      99,
					Name:     "stray",
					IsIndex:  true,
					DataType: telem.TimestampT,
				})).To(Succeed())
				image := MustSucceed(disk.Crash(fault.CrashConfig{}))
				// Cesium writes meta.json to a temporary file and renames it, with no
				// Sync. A directory sync makes the rename durable, not the bytes.
				info := MustSucceed(image.Stat("cesium/99/meta.json"))
				Expect(info.Size()).To(BeZero())
			},
		)

		It("Should make nothing durable on a Sub of a directory that exists",
			func(ctx SpecContext) {
				e := MustSucceed(openEngine(ctx, disk, true))
				DeferCleanup(func() { Expect(e.Close()).To(Succeed()) })
				f := MustSucceed(disk.Open("kv/probe", os.O_CREATE|os.O_WRONLY))
				MustSucceed(f.Write([]byte("taco")))
				Expect(f.Close()).To(Succeed())
				dir := strconv.FormatUint(uint64(e.clock.Key().StorageKey()), 10)
				MustSucceed(e.ts.Sub(dir))
				image := MustSucceed(disk.Crash(fault.CrashConfig{}))
				Expect(image.Exists("kv/probe")).To(BeFalse())
			},
		)
	})

	It("Should pass the audit of a crash that lost unsynced samples, under Partial",
		func(ctx SpecContext) {
			e := MustSucceed(openCrashEngine(ctx, disk))
			DeferCleanup(func() { Expect(e.Close()).To(Succeed()) })
			data, scratch := createPair(ctx, e, 1)
			// The creates are durable, as a Partial crash can keep them. No delete
			// runs, so only the unsynced samples can differ.
			disk.Writeback()
			Expect(writePair(ctx, e, 1, data, scratch, []byte("burrito"))).To(Succeed())
			var failed []string
			for key := range uint64(64) {
				image := MustSucceed(disk.Crash(fault.CrashConfig{
					RunKey:            key,
					DataSurvivalRate:  0.75,
					TornRate:          0.25,
					EntrySurvivalRate: 0.75,
				}))
				r, err := openCrashEngine(ctx, image)
				if err != nil {
					continue
				}
				// A failed read is an R4, not the audit.
				if _, err = r.Recovered(ctx); err == nil {
					if err = r.Audit(ctx); err != nil {
						failed = append(
							failed,
							strconv.FormatUint(key, 10)+": "+err.Error(),
						)
					}
				}
				Expect(r.Close()).To(Succeed())
			}
			Expect(failed).To(BeEmpty())
		},
	)

	It("Should break a rule when a recovery loses a commit that Commit wrote back",
		func(ctx SpecContext) {
			lost := 0
			result, err := crash.Run(ctx, crash.Config{
				Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
					if image, ok := fs.(*fault.FS); ok && loseNewest(image) {
						lost++
					}
					e, err := openEngine(ctx, fs, true)
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
			var first *crash.Violation
			Expect(errors.As(err, &first)).To(BeTrue(), "no violation: %v", err)
			Expect(lost).To(BeNumerically(">", 0))
			Expect(violations(result)).To(
				ContainElement(ContainSubstring("commit")),
				"%d recoveries lost a written-back commit",
				lost,
			)
		},
	)

	It("Should list the violations of an exhaustive run in the order of the events",
		func(ctx SpecContext) {
			result, err := crash.Run(ctx, crash.Config{
				Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
					e, err := openEngine(ctx, fs, false)
					if err != nil {
						return nil, err
					}
					if _, ok := fs.(*fault.FS); ok {
						return e, nil
					}
					return liveLoss{e}, nil
				},
				Commits:    4,
				Stride:     1,
				Profile:    crash.Strict,
				Exhaustive: true,
			})
			var first *crash.Violation
			Expect(errors.As(err, &first)).To(BeTrue())
			events := make([]int, len(result.Violations))
			for i, v := range result.Violations {
				events[i] = v.Event
			}
			Expect(slices.IsSorted(events)).To(BeTrue(), "events: %v", events)
		},
	)

	// The plain violations come from a removal that no sync makes durable, so they stay
	// when the record delete comes first. In the durable mode, a crash between the two
	// deletes then keeps the Cesium channel instead of losing the samples of d<k>.
	DescribeTable("Should find the violations of a delete whose record goes first",
		func(ctx SpecContext, dirsDurable bool, want []string) {
			result, err := crash.Run(ctx, crash.Config{
				Open: func(ctx context.Context, fs xfs.FS) (crash.Engine, error) {
					e, err := openEngine(ctx, fs, dirsDurable)
					if err != nil {
						return nil, err
					}
					return recordFirst{e}, nil
				},
				Commits:    4,
				Stride:     1,
				Profile:    crash.Strict,
				Exhaustive: true,
			})
			var first *crash.Violation
			Expect(errors.As(err, &first)).To(BeTrue(), "no violation: %v", err)
			Expect(violations(result)).To(ConsistOf(want))
		},
		Entry("plain", false, orphans(3, 5, 7, 9)),
		Entry("with durable Cesium directories", true, slices.Concat(
			emptyMeta("the engine did not recover", 1, 4, 6, 8),
			emptyMeta("the engine did not recover after a crash in recovery", 1, 2),
			[]string{
				"R4: the engine did not recover: data_type: required\nname: required\n" +
					"index: non-indexed channel must have an index",
			},
			orphans(3, 5, 7, 9),
		)),
	)
})
