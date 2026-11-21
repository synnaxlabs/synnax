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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var errCommit = errors.New("commit failed")

var _ = Describe("Crash adapter, kept data", func() {
	var (
		disk *fault.FS
		e    *crashEngine
	)
	BeforeEach(func(ctx SpecContext) {
		disk = MustSucceed(fault.New(fault.Config{}))
		e = MustSucceed(openCrashEngine(ctx, disk))
		DeferCleanup(func() {
			if e != nil {
				Expect(e.Close()).To(Succeed())
			}
		})
	})
	restart := func(ctx context.Context) {
		GinkgoHelper()
		Expect(e.Close()).To(Succeed())
		e = MustSucceed(openCrashEngine(ctx, disk))
	}
	// write creates the channel d<id> and writes payload as its samples at second id.
	write := func(ctx context.Context, id uint64, payload []byte) channel.Channel {
		GinkgoHelper()
		data := e.samples("d" + strconv.FormatUint(id, 10))
		Expect(e.create(ctx, &data)).To(Succeed())
		Expect(e.cesium.Write(ctx, telem.TimeStamp(id)*telem.SecondTS, telem.MultiFrame(
			[]cesium.ChannelKey{
				cesium.ChannelKey(e.clock.Key()),
				cesium.ChannelKey(data.Key()),
			},
			[]telem.Series{stamps(id, len(payload)), telem.NewSeries(payload)},
		))).To(Succeed())
		return data
	}
	// loseRecord deletes the record of ch and nothing else, as a lost Pebble tail does.
	loseRecord := func(ctx context.Context, ch channel.Channel) {
		GinkgoHelper()
		Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.NewDelete[channel.Key, channel.Channel]().
				Where(gorp.MatchKeys[channel.Key, channel.Channel](ch.Key())).
				Exec(ctx, tx)
		})).To(Succeed())
	}
	stored := func(key channel.Key) bool {
		GinkgoHelper()
		return MustSucceed(e.ts.Exists(strconv.FormatUint(uint64(key), 10)))
	}
	remove := func(ctx context.Context, key channel.Key) error {
		return e.db.WithTx(ctx, func(tx gorp.Tx) error {
			return e.svc.NewWriter(tx).Delete(ctx, key, false)
		})
	}

	It("Should keep a channel whose delete failed to commit, and clear its intent",
		func(ctx SpecContext) {
			data := write(ctx, 1, []byte("taco"))
			Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := e.svc.NewWriter(tx).
					Delete(ctx, data.Key(), false); err != nil {
					return err
				}
				gorp.BeforeCommit(tx, func(context.Context) error { return errCommit })
				return nil
			})).To(MatchError(errCommit))
			restart(ctx)
			Expect(e.read(ctx, data)).To(Equal([]byte("taco")))
			Expect((&deleteCases{}).files(disk)).
				ToNot(ContainElement(cesium.ChannelKey(data.Key())))
			Expect(e.cesium.Write(ctx, 2*telem.SecondTS, telem.MultiFrame(
				[]cesium.ChannelKey{
					cesium.ChannelKey(e.clock.Key()),
					cesium.ChannelKey(data.Key()),
				},
				[]telem.Series{stamps(2, 2), telem.NewSeries([]byte("ab"))},
			))).To(Succeed())
			Expect(e.read(ctx, data)).To(Equal([]byte("tacoab")))
		},
	)

	It("Should keep the samples of a kept channel that a delete names",
		func(ctx SpecContext) {
			data := write(ctx, 1, []byte("taco"))
			loseRecord(ctx, data)
			restart(ctx)
			Expect(stored(data.Key())).To(BeTrue())
			Expect(remove(ctx, data.Key())).To(Succeed())
			Expect(stored(data.Key())).To(BeTrue())
			restart(ctx)
			Expect(stored(data.Key())).To(BeTrue())
		},
	)

	It("Should open a kept data channel whose index a delete removed",
		func(ctx SpecContext) {
			data := write(ctx, 1, []byte("taco"))
			loseRecord(ctx, data)
			Expect(remove(ctx, e.clock.Key())).To(Succeed())
			restart(ctx)
			Expect(stored(data.Key())).To(BeTrue())
		},
	)

	It("Should give a new channel no key that storage keeps", func(ctx SpecContext) {
		data := write(ctx, 1, []byte("taco"))
		kept := []channel.Key{e.clock.Key(), data.Key()}
		Expect(e.Close()).To(Succeed())
		e = nil
		// The metadata store comes back empty, as after an undo.
		Expect(disk.Remove("kv")).To(Succeed())
		e = MustSucceed(openCrashEngine(ctx, disk))
		fresh := e.samples("d2")
		Expect(e.create(ctx, &fresh)).To(Succeed())
		Expect(kept).ToNot(ContainElement(e.clock.Key()))
		Expect(kept).ToNot(ContainElement(fresh.Key()))
		Expect(e.read(ctx, fresh)).To(BeEmpty())
		for _, k := range kept {
			Expect(stored(k)).To(BeTrue())
		}
	})
})
