// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/doctor"
	pkgdoctor "github.com/synnaxlabs/synnax/pkg/doctor"
	"github.com/synnaxlabs/synnax/pkg/storage"
	. "github.com/synnaxlabs/x/testutil"
)

// execute runs the doctor command against dir with the given flags, returning what it
// wrote to standard output and the error it exited with.
func execute(ctx context.Context, dir string, flags ...string) (string, error) {
	GinkgoHelper()
	var out bytes.Buffer
	viper.Reset()
	// Cmd is a package-level command, so flag values set by an earlier run persist.
	doctor.Cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if slice, isSlice := f.Value.(pflag.SliceValue); isSlice {
			Expect(slice.Replace(nil)).To(Succeed())
		} else {
			Expect(f.Value.Set(f.DefValue)).To(Succeed())
		}
		f.Changed = false
	})
	doctor.Cmd.SetOut(&out)
	doctor.Cmd.SetErr(&bytes.Buffer{})
	doctor.Cmd.SetArgs(append([]string{"--data", dir}, flags...))
	err := doctor.Cmd.ExecuteContext(ctx)
	return out.String(), err
}

var _ = Describe("Cmd", func() {
	var dir string

	BeforeEach(func(ctx SpecContext) {
		dir = MustSucceed(os.MkdirTemp("", "doctor-cmd-test-*"))
		DeferCleanup(func() { Expect(os.RemoveAll(dir)).To(Succeed()) })
		layer := MustSucceed(storage.OpenLayer(ctx, storage.LayerConfig{
			Dirname: dir,
		}))
		Expect(layer.Close()).To(Succeed())
	})

	It("Should write a text report", func(ctx SpecContext) {
		out, err := execute(ctx, dir)
		Expect(err).ToNot(HaveOccurred())
		Expect(out).To(ContainSubstring("directory"))
		Expect(out).To(ContainSubstring("no problems found"))
	})

	It("Should write a JSON report", func(ctx SpecContext) {
		out, err := execute(ctx, dir, "--json")
		Expect(err).ToNot(HaveOccurred())
		var report pkgdoctor.Report
		Expect(json.Unmarshal([]byte(out), &report)).To(Succeed())
		Expect(report.Dirname).To(Equal(dir))
		Expect(report.Findings).To(BeEmpty())
	})

	It("Should skip a store when told to", func(ctx SpecContext) {
		out, err := execute(ctx, dir, "--json", "--skip-kv")
		Expect(err).ToNot(HaveOccurred())
		var report pkgdoctor.Report
		Expect(json.Unmarshal([]byte(out), &report)).To(Succeed())
		Expect(report.KV).To(BeNil())
		Expect(report.TS).ToNot(BeNil())
	})

	It("Should return ErrProblems when the data holds errors", func(
		ctx SpecContext,
	) {
		layer := MustSucceed(storage.OpenLayer(ctx, storage.LayerConfig{
			Dirname: dir,
		}))
		key := append([]byte("gorp.Label"), 0, 0, 0, 1)
		Expect(layer.KV.Set(ctx, key, []byte{0xff, 0xff, 0xff})).To(Succeed())
		Expect(layer.Close()).To(Succeed())
		out, err := execute(ctx, dir)
		Expect(err).To(MatchError(doctor.ErrProblems))
		Expect(out).To(ContainSubstring("kv.decode"))
		Expect(out).To(ContainSubstring("1 errors, 1 warnings"))
	})

	It("Should fail when the data directory does not exist", func(ctx SpecContext) {
		Expect(execute(ctx, "/nope/not/here")).Error().To(
			MatchError(ContainSubstring("does not exist")),
		)
	})
})
