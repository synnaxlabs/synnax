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
	"encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/doctor"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Key-value checks", func() {
	Describe("Counters", func() {
		It("Should report a counter behind the keys it guards", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			createChannel(ctx, c, channel.Channel{
				Name:     "time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			key := c.Cluster.HostKey().String() +
				".distribution.channel.leasedCounter"
			Expect(c.Storage.KV.Set(ctx, []byte(key), make([]byte, 8))).To(Succeed())
			r := run(ctx, c)
			found := findings(r, doctor.CheckCounter)
			Expect(found).To(HaveLen(1))
			Expect(found[0].Severity).To(Equal(doctor.SeverityError))
			Expect(found[0].Message).To(Equal(
				"counter for leased channel keys is behind",
			))
			Expect(found[0].Subject).To(Equal(key))
		})

		It("Should accept a counter at the highest issued key", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			ch := createChannel(ctx, c, channel.Channel{
				Name:     "time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			key := c.Cluster.HostKey().String() +
				".distribution.channel.leasedCounter"
			value := make([]byte, 8)
			binary.LittleEndian.PutUint64(value, uint64(ch.LocalKey))
			Expect(c.Storage.KV.Set(ctx, []byte(key), value)).To(Succeed())
			r := run(ctx, c)
			Expect(findings(r, doctor.CheckCounter)).To(BeEmpty())
		})
	})

	Describe("Staging entries", func() {
		It("Should report leftover task migration entries", func(ctx SpecContext) {
			c := createCore(ctx)
			Expect(c.Storage.KV.Set(
				ctx,
				[]byte("sy_task_legacy_key/12"),
				[]byte("x"),
			)).To(Succeed())
			r := run(ctx, c)
			found := findings(r, doctor.CheckStaging)
			Expect(found).To(HaveLen(1))
			Expect(found[0].Severity).To(Equal(doctor.SeverityWarning))
		})
	})

	Describe("Migration state", func() {
		It("Should report a table that has not applied every migration", func(
			ctx SpecContext,
		) {
			c := createCore(ctx)
			createChannel(ctx, c, channel.Channel{
				Name:     "time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			Expect(c.Storage.KV.Delete(
				ctx,
				[]byte(gorp.MigrationKeyPrefix+"Channel"),
			)).To(Succeed())
			r := run(ctx, c)
			found := findings(r, doctor.CheckMigrationState)
			Expect(found).ToNot(BeEmpty())
			Expect(found[0].Message).To(ContainSubstring(
				"table has not applied every migration",
			))
			Expect(found[0].Subject).To(HavePrefix("Channel: "))
		})

		It("Should not report a table with no entries", func(ctx SpecContext) {
			c := createCore(ctx)
			Expect(c.Storage.KV.Delete(
				ctx,
				[]byte(gorp.MigrationKeyPrefix+"Channel"),
			)).To(Succeed())
			r := run(ctx, c)
			Expect(findings(r, doctor.CheckMigrationState)).To(BeEmpty())
		})
	})
})
