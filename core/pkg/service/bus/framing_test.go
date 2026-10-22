// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bus_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/bus"
	"github.com/synnaxlabs/synnax/pkg/service/library"
)

func syncFraming(edit func(*bus.SyncFraming)) bus.Framing {
	s := bus.SyncFraming{Sync: "AA55", LengthOffset: 2, LengthSize: 1}
	edit(&s)
	return bus.Framing{Variant: s}
}

var _ = Describe("Framing", func() {
	DescribeTable("CheckFraming accepts",
		func(f bus.Framing) { Expect(bus.CheckFraming(f)).To(Succeed()) },
		Entry("a delimiter framing", bus.Framing{
			Variant: bus.DelimiterFraming{Delimiter: "\n"},
		}),
		Entry("a COBS framing", bus.Framing{Variant: bus.CobsFraming{}}),
		Entry("a sync framing", syncFraming(func(*bus.SyncFraming) {})),
		Entry("a sync framing with no sync sequence", syncFraming(
			func(s *bus.SyncFraming) { s.Sync, s.LengthOffset = "", 0 },
		)),
		Entry("a max length that holds the header and checksum", syncFraming(
			func(s *bus.SyncFraming) {
				s.Checksum = bus.ChecksumCrc32
				s.MaxLength = 7
			},
		)),
	)

	DescribeTable("CheckFraming rejects",
		func(f bus.Framing, msg string) {
			Expect(bus.CheckFraming(f)).To(MatchError(msg))
		},
		Entry("an odd number of hex digits", syncFraming(
			func(s *bus.SyncFraming) { s.Sync = "AA5" },
		), "sync: must be pairs of hex digits, such as AA55"),
		Entry("a sync sequence that is not hex", syncFraming(
			func(s *bus.SyncFraming) { s.Sync = "ZZ" },
		), "sync: must be pairs of hex digits, such as AA55"),
		Entry("a length size of 3 bytes", syncFraming(
			func(s *bus.SyncFraming) { s.LengthSize = 3 },
		), "length_size: must be 1, 2, or 4 bytes"),
		Entry("a length field that overlaps the sync sequence", syncFraming(
			func(s *bus.SyncFraming) { s.LengthOffset = 1 },
		), "length_offset: must not overlap the sync sequence"),
		Entry("a length field past the largest frame", syncFraming(
			func(s *bus.SyncFraming) { s.LengthOffset = bus.MaxFrameSize },
		), "length_offset: must leave room for the length field within 65536 bytes"),
		Entry("a max length that cannot hold the checksum", syncFraming(
			func(s *bus.SyncFraming) {
				s.Checksum = bus.ChecksumCrc16Modbus
				s.MaxLength = 4
			},
		), "max_length: must be 0 or at least 5 bytes"),
	)

	Describe("Resolver", func() {
		var (
			resolver bus.Resolver
			lib      library.Library
			task     uuid.UUID
		)
		BeforeEach(func(ctx SpecContext) {
			resolver = bus.Resolver{Stamper: library.Stamper{DB: db, Ontology: otg}}
			lib = library.Library{Name: "Bench"}
			task = uuid.New()
			createLibrary(ctx, &lib, task)
		})

		It(
			"Should accept a stream read config with a valid framing",
			func(ctx SpecContext) {
				cfg := bus.StreamReadConfig{
					ReadConfig: bus.ReadConfig{
						Reference: library.Reference{Library: &lib.Key},
					},
					Framing: syncFraming(func(*bus.SyncFraming) {}),
				}
				Expect(resolver.StreamRead(ctx, tx, task, &cfg)).To(Succeed())
			},
		)

		It(
			"Should reject a stream read config with a bad framing",
			func(ctx SpecContext) {
				cfg := bus.StreamReadConfig{
					ReadConfig: bus.ReadConfig{
						Reference: library.Reference{Library: &lib.Key},
					},
					Framing: syncFraming(func(s *bus.SyncFraming) { s.LengthSize = 3 }),
				}
				Expect(resolver.StreamRead(ctx, tx, task, &cfg)).
					To(MatchError("framing.length_size: must be 1, 2, or 4 bytes"))
			},
		)

		It(
			"Should reject a stream write config with a bad framing",
			func(ctx SpecContext) {
				cfg := bus.StreamWriteConfig{
					WriteConfig: bus.WriteConfig{
						Reference: library.Reference{Library: &lib.Key},
					},
					Framing: syncFraming(func(s *bus.SyncFraming) { s.Sync = "ZZ" }),
				}
				Expect(resolver.StreamWrite(ctx, tx, task, &cfg)).
					To(MatchError("framing.sync: must be pairs of hex digits, such as AA55"))
			},
		)

		It(
			"Should accept a stream write config with a valid framing",
			func(ctx SpecContext) {
				cfg := bus.StreamWriteConfig{
					WriteConfig: bus.WriteConfig{
						Reference: library.Reference{Library: &lib.Key},
					},
					Framing: bus.Framing{Variant: bus.SlipFraming{}},
				}
				Expect(resolver.StreamWrite(ctx, tx, task, &cfg)).To(Succeed())
			},
		)
	})
})
