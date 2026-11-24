// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package repair_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/cmd/repair"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// logName is the file of log 1 of the telemetry of a data directory.
const logName = "cesium/0000000000000001.log"

var _ = Describe("Repair", func() {
	var dir string
	pointerLog := storage.LayerConfig{PointerLogEnabled: new(true)}
	openLayer := func(ctx context.Context) (*storage.Layer, error) {
		return storage.OpenLayer(ctx, storage.LayerConfig{
			Dirname:           dir,
			Version:           "0.0.0",
			PointerLogEnabled: new(true),
		})
	}
	BeforeEach(func(ctx SpecContext) {
		dir = GinkgoT().TempDir()
		l := MustSucceed(openLayer(ctx))
		stop := StartRun(l.Run)
		Expect(l.TS.CreateChannel(ctx, cesium.Channel{
			Key:      1,
			Name:     "time",
			IsIndex:  true,
			DataType: telem.TimestampT,
		})).To(Succeed())
		for s := range telem.TimeStamp(4) {
			Expect(l.TS.WriteSeries(
				ctx,
				1,
				(10*s+1)*telem.SecondTS,
				telem.NewSeriesSecondsTSV(10*s+1, 10*s+2, 10*s+3),
			)).To(Succeed())
		}
		stop()
		Expect(l.Close()).To(Succeed())
	})
	logPath := func() string { return filepath.Clean(filepath.Join(dir, logName)) }
	// spoil changes a byte of a frame of log 1 that valid frames follow. A frame starts
	// with the magic CPLF at a block of 4096 bytes from offset 8192 on.
	spoil := func() {
		GinkgoHelper()
		b := MustSucceed(os.ReadFile(logPath()))
		var frames []int
		for off := 8192; off+4 <= len(b); off += 4096 {
			if string(b[off:off+4]) == "CPLF" {
				frames = append(frames, off)
			}
		}
		Expect(len(frames)).To(BeNumerically(">=", 3))
		b[frames[len(frames)/2]+4] ^= 0xFF
		Expect(os.WriteFile(logPath(), b, 0o644)).To(Succeed())
	}
	// run runs the command on dir, and returns its output, or its error alone.
	run := func(layer storage.LayerConfig, in io.Reader, args ...string) (string, error) {
		var out bytes.Buffer
		cmd := repair.NewCmd("0.0.0", func() alamos.Instrumentation {
			return alamos.Instrumentation{}
		}, layer)
		cmd.SetIn(in)
		cmd.SetOut(&out)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(append(args, "--data", dir))
		if err := cmd.Execute(); err != nil {
			return "", err
		}
		return out.String(), nil
	}

	It("Should say that there is nothing to repair on the old storage layout", func() {
		Expect(run(storage.LayerConfig{}, strings.NewReader(""))).Error().
			To(MatchError("nothing to repair: this Core uses the old storage layout"))
	})

	It("Should say that the database opens when nothing is damaged", func() {
		Expect(MustSucceed(run(pointerLog, strings.NewReader("")))).
			To(ContainSubstring("nothing to repair: the database opens"))
	})

	It("Should refuse with no terminal to ask on, and name --yes", func() {
		spoil()
		before := MustSucceed(os.ReadFile(logPath()))
		in := MustSucceed(os.Open(logPath()))
		defer func() { Expect(in.Close()).To(Succeed()) }()
		Expect(run(pointerLog, in)).Error().To(MatchError(ContainSubstring("--yes")))
		Expect(os.ReadFile(logPath())).To(Equal(before))
	})

	It("Should stop when the answer is not yes, and change no file", func() {
		spoil()
		before := MustSucceed(os.ReadFile(logPath()))
		out := MustSucceed(run(pointerLog, strings.NewReader("n\n")))
		Expect(out).To(ContainSubstring("Damaged log frames"))
		Expect(out).To(ContainSubstring("Stopped the repair. No file changed."))
		Expect(os.ReadFile(logPath())).To(Equal(before))
	})

	DescribeTable("Should repair, so that the Core opens the data",
		func(ctx SpecContext, in string, args ...string) {
			spoil()
			Expect(openLayer(ctx)).Error().
				To(MatchError(ContainSubstring("holds a valid frame")))
			out := MustSucceed(run(pointerLog, strings.NewReader(in), args...))
			Expect(out).To(ContainSubstring("Repaired."))
			Expect(out).To(ContainSubstring(logName[len("cesium/"):] + ".damaged"))
			Expect(MustSucceed(openLayer(ctx)).Close()).To(Succeed())
		},
		Entry("on a yes", "y\n"),
		Entry("with --yes", "", "--yes"),
	)

	It("Should refuse while a Core holds the data", func(ctx SpecContext) {
		l := MustSucceed(openLayer(ctx))
		defer func() { Expect(l.Close()).To(Succeed()) }()
		Expect(run(pointerLog, strings.NewReader(""), "--yes")).Error().To(
			MatchError(ContainSubstring("Failed to acquire lock on storage directory")),
		)
	})
})
