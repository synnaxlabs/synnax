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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Resolver", func() {
	var (
		resolver bus.Resolver
		lib      library.Library
		status   library.MessageEntry
		command  library.MessageEntry
		task     uuid.UUID
	)
	BeforeEach(func(ctx SpecContext) {
		resolver = bus.Resolver{Stamper: library.Stamper{DB: db, Ontology: otg}}
		status = canMessage(
			"Status",
			0x100,
			binaryField("rpm", 0),
			binaryField("temp", 8),
		)
		command = canMessage("Command", 0x200, binaryField("throttle", 0))
		lib = library.Library{
			Name: "Engine",
			Entries: []library.Entry{
				{Variant: status},
				{Variant: command},
				{Variant: library.EnumEntry{
					BaseEntry: library.BaseEntry{Key: uuid.New(), Name: "Mode"},
				}},
			},
		}
		task = uuid.New()
		createLibrary(ctx, &lib, task)
	})

	readConfig := func(messages ...bus.ReadMessage) bus.ReadConfig {
		return bus.ReadConfig{
			Library:  lib.Key,
			Messages: messages,
		}
	}

	Describe("Read", func() {
		It("Should stamp the library hash into the config", func(ctx SpecContext) {
			cfg := readConfig(bus.ReadMessage{
				Message: status.Key,
				Fields: []bus.ReadField{
					{Field: fieldKey(status.Fields[1]), Channel: 2},
				},
			})
			Expect(resolver.Read(ctx, tx, task, &cfg)).To(Succeed())
			Expect(cfg.LibraryHash).To(Equal(MustSucceed(library.Hash(lib))))
		})

		It("Should reject a message that is not in the library", func(ctx SpecContext) {
			missing := uuid.New()
			cfg := readConfig(
				bus.ReadMessage{Message: status.Key},
				bus.ReadMessage{Message: missing},
			)
			Expect(resolver.Read(ctx, tx, task, &cfg)).To(MatchError(ContainSubstring(
				"messages.1.message: message " + missing.String() +
					" is not in the library: validation error",
			)))
		})

		It("Should reject an enum entry used as a message", func(ctx SpecContext) {
			enum := lib.Entries[2].Variant.(library.EnumEntry)
			cfg := readConfig(bus.ReadMessage{Message: enum.Key})
			Expect(resolver.Read(ctx, tx, task, &cfg)).
				To(MatchError(ContainSubstring("messages.0.message")))
		})

		It("Should reject a field of another message", func(ctx SpecContext) {
			other := fieldKey(command.Fields[0])
			cfg := readConfig(bus.ReadMessage{
				Message: status.Key,
				Fields: []bus.ReadField{
					{Field: fieldKey(status.Fields[0])},
					{Field: other},
				},
			})
			Expect(resolver.Read(ctx, tx, task, &cfg)).To(MatchError(ContainSubstring(
				"messages.0.fields.1.field: field " + other.String() +
					" is not in message Status",
			)))
		})

		It(
			"Should reject a config whose library does not exist",
			func(ctx SpecContext) {
				cfg := bus.ReadConfig{Library: uuid.New()}
				Expect(resolver.Read(ctx, tx, task, &cfg)).
					To(MatchError(ContainSubstring("does not exist")))
			},
		)
	})

	Describe("Write", func() {
		It("Should stamp the library hash into the config", func(ctx SpecContext) {
			cfg := bus.WriteConfig{
				Library: lib.Key,
				Messages: []bus.WriteMessage{{
					Message: command.Key,
					Fields: []bus.WriteField{
						{Field: fieldKey(command.Fields[0]), Channel: 3},
					},
				}},
			}
			Expect(resolver.Write(ctx, tx, task, &cfg)).To(Succeed())
			Expect(cfg.LibraryHash).To(Equal(MustSucceed(library.Hash(lib))))
		})

		It("Should reject a field that is not in the message", func(ctx SpecContext) {
			missing := uuid.New()
			cfg := bus.WriteConfig{
				Library: lib.Key,
				Messages: []bus.WriteMessage{{
					Message: command.Key,
					Fields:  []bus.WriteField{{Field: missing}},
				}},
			}
			Expect(resolver.Write(ctx, tx, task, &cfg)).To(MatchError(ContainSubstring(
				"messages.0.fields.0.field: field " + missing.String() +
					" is not in message Command",
			)))
		})
	})
})
