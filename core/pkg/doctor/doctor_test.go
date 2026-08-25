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
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/doctor"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Doctor", func() {
	Describe("Config", func() {
		It("Should reject a missing data directory name", func(ctx SpecContext) {
			Expect(doctor.Run(ctx, doctor.Config{})).Error().To(
				MatchError(ContainSubstring("dirname")),
			)
		})

		It("Should reject a directory that does not exist", func(ctx SpecContext) {
			Expect(doctor.Run(ctx, doctor.Config{Dirname: "/nope/not/here"})).Error().
				To(MatchError(validate.ErrValidation))
		})
	})

	Describe("A healthy Core", func() {
		It("Should report no problems", func(ctx SpecContext) {
			c := createCore(ctx)
			ch := createChannel(ctx, c, channel.Channel{
				Name:     "time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			Expect(ch.Key()).ToNot(BeZero())
			r := run(ctx, c)
			Expect(r.Errors()).To(Equal(0))
			Expect(r.Findings).To(BeEmpty())
		})

		It("Should summarize both stores", func(ctx SpecContext) {
			c := createCore(ctx)
			host := c.Cluster.HostKey()
			ch := createChannel(ctx, c, channel.Channel{
				Name:     "time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			Expect(ch.Key()).ToNot(BeZero())
			r := run(ctx, c)
			Expect(r.KV.Entries).To(BeNumerically(">", 0))
			Expect(r.KV.Bytes).To(BeNumerically(">", 0))
			Expect(r.KV.Cluster.HostKey).To(Equal(host))
			Expect(r.KV.Cluster.Nodes).To(HaveLen(1))
			Expect(r.KV.Cluster.Nodes[0].State).To(Equal("healthy"))
			Expect(r.KV.Buckets).ToNot(BeEmpty())
			Expect(r.TS.Channels).To(HaveLen(1))
		})
	})

	Describe("Skipping work", func() {
		It("Should skip the time-series store", func(ctx SpecContext) {
			r := run(ctx, createCore(ctx), doctor.Config{TSDisabled: true})
			Expect(r.TS).To(BeNil())
			Expect(r.KV).ToNot(BeNil())
		})

		It("Should skip the key-value store", func(ctx SpecContext) {
			r := run(ctx, createCore(ctx), doctor.Config{KVDisabled: true})
			Expect(r.KV).To(BeNil())
			Expect(r.KVUnavailable).To(BeEmpty())
			Expect(r.TS).ToNot(BeNil())
		})

		It("Should report a key-value store another process holds", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			r := MustSucceed(doctor.Run(ctx, doctor.Config{Dirname: c.Dirname}))
			Expect(c.Close()).To(Succeed())
			Expect(r.KV).To(BeNil())
			Expect(r.KVUnavailable).To(ContainSubstring("lock"))
			Expect(r.TS).ToNot(BeNil())
		})
	})

	Describe("Progress", func() {
		It("Should report progress through every phase", func(ctx SpecContext) {
			phases := make(map[doctor.Phase]int)
			run(ctx, createCore(ctx), doctor.Config{
				Progress: func(phase doctor.Phase, done, total int) {
					Expect(done).To(BeNumerically("<=", total))
					phases[phase] = done
				},
			})
			Expect(phases).To(HaveKey(doctor.PhaseKV))
		})
	})

	Describe("Rendering", func() {
		It("Should render the report as text", func(ctx SpecContext) {
			var buf bytes.Buffer
			r := run(ctx, createCore(ctx))
			Expect(doctor.Render(&buf, r, false)).To(Succeed())
			Expect(buf.String()).To(ContainSubstring("directory"))
			Expect(buf.String()).To(ContainSubstring("no problems found"))
		})

		It("Should render the per-channel table when verbose", func(
			ctx SpecContext,
		) {
			var buf bytes.Buffer
			c := createCore(ctx)
			ch := createChannel(ctx, c, channel.Channel{
				Name:     "verbose_time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			Expect(ch.Key()).ToNot(BeZero())
			r := run(ctx, c)
			Expect(doctor.Render(&buf, r, true)).To(Succeed())
			Expect(buf.String()).To(ContainSubstring("verbose_time"))
		})

		It("Should render the report as JSON", func(ctx SpecContext) {
			var buf bytes.Buffer
			r := run(ctx, createCore(ctx))
			Expect(doctor.RenderJSON(&buf, r)).To(Succeed())
			var decoded doctor.Report
			Expect(json.Unmarshal(buf.Bytes(), &decoded)).To(Succeed())
			Expect(decoded.Dirname).To(Equal(r.Dirname))
			Expect(decoded.KV.Entries).To(Equal(r.KV.Entries))
		})
	})

	Describe("Unknown keys", func() {
		It("Should report a key that belongs to no known table", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			Expect(c.Storage.KV.Set(ctx, []byte("gorp.Mystery1"), []byte("x"))).
				To(Succeed())
			r := run(ctx, c)
			found := findings(r, doctor.CheckUnknownPrefix)
			Expect(found).To(HaveLen(1))
			Expect(found[0].Severity).To(Equal(doctor.SeverityWarning))
			Expect(found[0].Subject).To(Equal("gorp.Mystery1"))
		})
	})

	Describe("Undecodable entries", func() {
		It("Should report an entry no codec can decode", func(ctx SpecContext) {
			c := createCore(ctx)
			ch := createChannel(ctx, c, channel.Channel{
				Name:     "time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			Expect(ch.Key()).ToNot(BeZero())
			key := append([]byte("gorp.Label"), 0, 0, 0, 1)
			Expect(c.Storage.KV.Set(ctx, key, []byte{0xff, 0xff, 0xff})).To(Succeed())
			r := run(ctx, c)
			found := findings(r, doctor.CheckDecode)
			Expect(found).To(HaveLen(1))
			Expect(found[0].Severity).To(Equal(doctor.SeverityError))
			Expect(found[0].Message).To(ContainSubstring("Label"))
			Expect(r.Errors()).To(Equal(1))
		})
	})

	Describe("Cross-layer", func() {
		It("Should report a Cesium directory with no channel entry", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			ch := createChannel(ctx, c, channel.Channel{
				Name:     "orphan_storage",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](c.DB).
				Delete(ctx, ch.Key())).To(Succeed())
			r := run(ctx, c)
			found := findings(r, doctor.CheckChannelDir)
			Expect(found).To(HaveLen(1))
			Expect(found[0].Message).To(ContainSubstring("no channel entry"))
		})
	})

	Describe("A directory with no time-series store", func() {
		It("Should report no channels rather than failing", func(ctx SpecContext) {
			c := createCore(ctx)
			Expect(c.Close()).To(Succeed())
			Expect(os.RemoveAll(filepath.Join(c.Dirname, storage.TSDirName))).
				To(Succeed())
			r := MustSucceed(doctor.Run(ctx, doctor.Config{Dirname: c.Dirname}))
			Expect(r.TS.Channels).To(BeEmpty())
		})
	})
})
