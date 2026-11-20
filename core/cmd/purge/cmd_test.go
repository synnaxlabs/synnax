// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package purge_test

import (
	"bytes"
	"context"
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/cmd/purge"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Purge", func() {
	var (
		dir              string
		orphan, recorded channel.Key
	)
	openLayer := func(ctx context.Context) *storage.Layer {
		GinkgoHelper()
		return MustSucceed(storage.OpenLayer(ctx, storage.LayerConfig{
			Dirname: dir,
			Version: "0.0.0",
		}))
	}
	// createIndex creates a Cesium index channel with samples at key.
	createIndex := func(ctx context.Context, l *storage.Layer, key channel.Key) {
		GinkgoHelper()
		Expect(l.TS.CreateChannel(ctx, cesium.Channel{
			Key:      uint32(key),
			Name:     "time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		})).To(Succeed())
		w := MustSucceed(l.TS.OpenWriter(ctx, cesium.WriterConfig{
			Channels: []cesium.ChannelKey{uint32(key)},
			Clock:    uint32(key),
			Start:    10 * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.UnaryFrame(
			uint32(key),
			telem.NewSeriesSecondsTSV(10, 11, 12),
		), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
	}
	BeforeEach(func(ctx SpecContext) {
		dir = GinkgoT().TempDir()
		orphan = distchannel.NewKey(1, 60_000)
		ch := channel.Channel{
			Name:        "recorded",
			Leaseholder: 1,
			LocalKey:    60_001,
			IsIndex:     true,
			DataType:    telem.TimestampT,
		}
		recorded = ch.Key()
		l := openLayer(ctx)
		createIndex(ctx, l, orphan)
		createIndex(ctx, l, recorded)
		Expect(gorp.NewCreate[channel.Key, channel.Channel]().
			Entry(&ch).
			Exec(ctx, l.Gorp)).To(Succeed())
		Expect(l.Close()).To(Succeed())
	})
	run := func(vsn string, args ...string) (string, error) {
		var out bytes.Buffer
		cmd := purge.NewCmd(vsn, func() alamos.Instrumentation {
			return alamos.Instrumentation{}
		})
		cmd.SetOut(&out)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(append(args, "--data", dir))
		err := cmd.Execute()
		return out.String(), err
	}

	It("Should list the channels that hold data with no record", func() {
		out := MustSucceed(run("0.0.0"))
		Expect(out).To(MatchRegexp(`^CHANNEL +BYTES\n%d +[1-9][0-9]*\n\n`, orphan))
		Expect(out).NotTo(ContainSubstring(fmt.Sprint(recorded)))
		Expect(out).To(ContainSubstring("synnax purge <channel>"))
	})

	It("Should remove the data of a listed channel", func(ctx SpecContext) {
		Expect(MustSucceed(run("0.0.0", fmt.Sprint(orphan)))).
			To(Equal(fmt.Sprintf("Removed the data of channel %d.\n", orphan)))
		Expect(MustSucceed(run("0.0.0"))).
			To(Equal("No channel holds data without a channel record.\n"))
		l := openLayer(ctx)
		DeferCleanup(l.Close)
		Expect(l.TS.RetrieveChannel(ctx, uint32(orphan))).Error().
			To(MatchError(cesium.ErrChannelNotFound))
		MustSucceed(l.TS.RetrieveChannel(ctx, uint32(recorded)))
	})

	It("Should refuse a channel that has a record", func() {
		Expect(run("0.0.0", fmt.Sprint(recorded))).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"channel %d has a channel record. To remove its data, delete the channel",
				recorded,
			)),
		))
	})

	It("Should refuse a channel that holds no data to purge", func() {
		Expect(run("0.0.0", fmt.Sprint(orphan), "70000")).Error().
			To(MatchError(query.ErrNotFound))
		Expect(MustSucceed(run("0.0.0"))).To(ContainSubstring(fmt.Sprint(orphan)))
	})

	It("Should refuse an argument that is not a channel key", func() {
		Expect(run("0.0.0", "twelve")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`"twelve" is not a channel key`)),
		))
	})

	It("Should refuse while a Core holds the data", func(ctx SpecContext) {
		l := openLayer(ctx)
		DeferCleanup(l.Close)
		Expect(run("0.0.0")).Error().
			To(MatchError(ContainSubstring("Failed to acquire lock on storage directory")))
	})

	It("Should refuse data that another Core version opened last", func() {
		Expect(run("0.62.0")).Error().To(MatchError(
			"Core 0.62.0 did not open the data last. Start Core 0.62.0 on the data, " +
				"stop it, and run the purge again.",
		))
	})
})
