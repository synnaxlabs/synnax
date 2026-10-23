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
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Resolver", func() {
	var (
		resolver bus.Resolver
		ic       icd.ICD
		status   icd.MessageEntry
		command  icd.MessageEntry
		rpm      icd.BinaryField
		temp     icd.BinaryField
		throttle icd.BinaryField
		task     uuid.UUID
	)
	BeforeEach(func(ctx SpecContext) {
		resolver = bus.Resolver{Stamper: icd.Stamper{DB: db, Ontology: otg}}
		rpm = binaryField("rpm", 0)
		temp = binaryField("temp", 8)
		throttle = binaryField("throttle", 0)
		status = canMessage("Status", 0x100, rpm, temp)
		command = canMessage("Command", 0x200, throttle)
		ic = icd.ICD{
			Name: "Engine",
			Entries: []icd.Entry{
				{Variant: status},
				{Variant: command},
				{Variant: icd.EnumEntry{
					BaseEntry: icd.BaseEntry{Key: uuid.New(), Name: "Mode"},
				}},
			},
		}
		task = uuid.New()
		createICD(ctx, &ic, task)
	})

	readConfig := func(messages ...bus.ReadMessage) bus.ReadConfig {
		return bus.ReadConfig{
			ICD:      &ic.Key,
			Messages: messages,
		}
	}

	Describe("Override", func() {
		It("Should take each nil dependency from the other resolver", func() {
			Expect(bus.Resolver{}.Override(resolver)).To(Equal(resolver))
		})

		It("Should keep a dependency that is already set", func() {
			Expect(resolver.Override(bus.Resolver{})).To(Equal(resolver))
		})
	})

	Describe("Validate", func() {
		It("Should accept a resolver with every dependency", func() {
			Expect(resolver.Validate()).To(Succeed())
		})

		It("Should reject a resolver missing the database", func() {
			Expect(bus.Resolver{
				Stamper: icd.Stamper{Ontology: otg},
			}.Validate()).To(MatchError("stamper.db: must be non-nil"))
		})

		It("Should reject a resolver missing the ontology", func() {
			Expect(bus.Resolver{
				Stamper: icd.Stamper{DB: db},
			}.Validate()).To(MatchError("stamper.ontology: must be non-nil"))
		})
	})

	Describe("Read", func() {
		It(
			"Should stamp the hash of the messages the config reads",
			func(ctx SpecContext) {
				cfg := readConfig(bus.ReadMessage{
					Message: status.Key,
					Fields: []bus.ReadField{
						{Field: temp.Key, Channel: 2},
					},
				})
				Expect(resolver.Read(ctx, tx, task, &cfg)).To(Succeed())
				Expect(cfg.ICDHash).To(Equal(MustSucceed(
					icd.TaskHash(ic, []icd.EntryKey{status.Key}),
				)))
			},
		)

		It("Should reject a message that is not in the ICD", func(ctx SpecContext) {
			missing := uuid.New()
			cfg := readConfig(
				bus.ReadMessage{Message: status.Key},
				bus.ReadMessage{Message: missing},
			)
			Expect(resolver.Read(ctx, tx, task, &cfg)).To(MatchError(ContainSubstring(
				"messages.1.message: message " + missing.String() +
					" is not in the ICD: validation error",
			)))
		})

		It("Should reject an enum entry used as a message", func(ctx SpecContext) {
			enum := ic.Entries[2].Variant.(icd.EnumEntry)
			cfg := readConfig(bus.ReadMessage{Message: enum.Key})
			Expect(resolver.Read(ctx, tx, task, &cfg)).
				To(MatchError(ContainSubstring("messages.0.message")))
		})

		It("Should reject a field of another message", func(ctx SpecContext) {
			other := throttle.Key
			cfg := readConfig(bus.ReadMessage{
				Message: status.Key,
				Fields: []bus.ReadField{
					{Field: rpm.Key},
					{Field: other},
				},
			})
			Expect(resolver.Read(ctx, tx, task, &cfg)).To(MatchError(ContainSubstring(
				"messages.0.fields.1.field: field " + other.String() +
					" is not in message Status",
			)))
		})

		It(
			"Should reject a config whose ICD does not exist",
			func(ctx SpecContext) {
				missing := uuid.New()
				cfg := bus.ReadConfig{ICD: &missing}
				Expect(
					resolver.Read(ctx, tx, task, &cfg),
				).To(MatchError(ContainSubstring(
					"icd: ICD " + missing.String() + " does not exist",
				)))
			},
		)

		It(
			"Should accept a config with no ICD and no messages",
			func(ctx SpecContext) {
				cfg := bus.ReadConfig{ICDHash: "stale"}
				Expect(resolver.Read(ctx, tx, task, &cfg)).To(Succeed())
				Expect(cfg.ICDHash).To(BeEmpty())
			},
		)

		It(
			"Should reject a config with messages but no ICD",
			func(ctx SpecContext) {
				cfg := bus.ReadConfig{
					Messages: []bus.ReadMessage{{Message: status.Key}},
				}
				Expect(
					resolver.Read(ctx, tx, task, &cfg),
				).To(MatchError(ContainSubstring(
					"icd: select an ICD before choosing messages",
				)))
			},
		)
	})

	Describe("Read indexes", func() {
		var (
			battery             icd.MessageEntry
			mode, volt, current icd.BinaryField
			temp1, temp2, humid icd.BinaryField
		)
		BeforeEach(func(ctx SpecContext) {
			mode = binaryField("mode", 0)
			volt = multiplexed(binaryField("volt", 8), mode, 0)
			current = multiplexed(binaryField("current", 16), mode, 0)
			temp1 = multiplexed(binaryField("temp1", 8), mode, 1, 2)
			temp2 = multiplexed(binaryField("temp2", 16), mode, 2, 1)
			humid = multiplexed(binaryField("humid", 24), mode, 1)
			battery = canMessage(
				"Battery", 0x300, mode, volt, current, temp1, temp2, humid,
			)
			ic.Entries = append(ic.Entries, icd.Entry{Variant: battery})
			task = uuid.New()
			createICD(ctx, &ic, task)
		})

		It("Should accept a plain message on one index", func(ctx SpecContext) {
			cfg := readConfig(bus.ReadMessage{
				Message: status.Key,
				Fields: []bus.ReadField{
					{Field: rpm.Key, Channel: 2, Index: 1},
					{Field: temp.Key, Channel: 3, Index: 1},
				},
			})
			Expect(resolver.Read(ctx, tx, task, &cfg)).To(Succeed())
		})

		It(
			"Should accept multiplexed fields on one index per group",
			func(ctx SpecContext) {
				cfg := readConfig(bus.ReadMessage{
					Message: battery.Key,
					Fields: []bus.ReadField{
						{Field: mode.Key, Channel: 2, Index: 1},
						{Field: volt.Key, Channel: 3, Index: 4},
						{Field: current.Key, Channel: 5, Index: 4},
						{Field: temp1.Key, Channel: 6, Index: 7},
						{Field: temp2.Key, Channel: 8, Index: 7},
						{Field: humid.Key, Channel: 9, Index: 10},
					},
				})
				Expect(resolver.Read(ctx, tx, task, &cfg)).To(Succeed())
			},
		)

		It(
			"Should reject fields that different multiplexor values select on one index",
			func(ctx SpecContext) {
				cfg := readConfig(bus.ReadMessage{
					Message: battery.Key,
					Fields: []bus.ReadField{
						{Field: volt.Key, Channel: 3, Index: 4},
						{Field: temp1.Key, Channel: 6, Index: 7},
						{Field: humid.Key, Channel: 9, Index: 7},
					},
				})
				Expect(resolver.Read(ctx, tx, task, &cfg)).To(MatchError(
					"messages.0.fields.2.index: fields temp1 and humid share index " +
						"7, but different multiplexor values select them: " +
						"validation error",
				))
			},
		)

		It(
			"Should reject a multiplexed field on the index of an always-present field",
			func(ctx SpecContext) {
				cfg := readConfig(bus.ReadMessage{
					Message: battery.Key,
					Fields: []bus.ReadField{
						{Field: mode.Key, Channel: 2, Index: 1},
						{Field: volt.Key, Channel: 3, Index: 1},
					},
				})
				Expect(resolver.Read(ctx, tx, task, &cfg)).To(MatchError(
					"messages.0.fields.1.index: fields mode and volt share index " +
						"1, but different multiplexor values select them: " +
						"validation error",
				))
			},
		)

		It("Should reject an index shared by two messages", func(ctx SpecContext) {
			cfg := readConfig(
				bus.ReadMessage{
					Message: status.Key,
					Fields:  []bus.ReadField{{Field: rpm.Key, Channel: 2, Index: 1}},
				},
				bus.ReadMessage{
					Message: battery.Key,
					Fields:  []bus.ReadField{{Field: mode.Key, Channel: 3, Index: 1}},
				},
			)
			Expect(resolver.Read(ctx, tx, task, &cfg)).To(MatchError(
				"messages.1.fields.0.index: index 1 already stamps message " +
					"Status: validation error",
			))
		})

		It(
			"Should accept fields of different groups with no index",
			func(ctx SpecContext) {
				cfg := readConfig(
					bus.ReadMessage{
						Message: status.Key,
						Fields:  []bus.ReadField{{Field: rpm.Key, Channel: 2}},
					},
					bus.ReadMessage{
						Message: battery.Key,
						Fields: []bus.ReadField{
							{Field: volt.Key, Channel: 3},
							{Field: humid.Key, Channel: 4},
						},
					},
				)
				Expect(resolver.Read(ctx, tx, task, &cfg)).To(Succeed())
			},
		)

		It("Should skip a disabled message", func(ctx SpecContext) {
			cfg := readConfig(
				bus.ReadMessage{
					Message: status.Key,
					Fields:  []bus.ReadField{{Field: rpm.Key, Channel: 2, Index: 1}},
				},
				bus.ReadMessage{
					Message:  battery.Key,
					Disabled: true,
					Fields: []bus.ReadField{
						{Field: volt.Key, Channel: 3, Index: 1},
						{Field: humid.Key, Channel: 4, Index: 1},
					},
				},
			)
			Expect(resolver.Read(ctx, tx, task, &cfg)).To(Succeed())
		})
	})

	Describe("Read replies", func() {
		var (
			reading icd.MessageEntry
			labeled icd.MessageEntry
			line    icd.MessageEntry
			value   icd.BinaryField
			level   icd.BinaryField
		)
		BeforeEach(func(ctx SpecContext) {
			value = binaryField("value", 0)
			reading = polled("Reading", "\\x01", value)
			level = binaryField("level", 8)
			labeled = canMessage("Level", 0x400, level)
			query := "LEVEL?"
			labeled.Query = &query
			lineQuery := "MEAS?"
			line = icd.MessageEntry{
				Key: uuid.New(), Name: "Line",
				Payload: icd.Payload{Variant: icd.TextPayload{
					Delimiter: ",",
				}},
				Query: &lineQuery,
			}
			ic.Entries = append(
				ic.Entries,
				icd.Entry{Variant: reading},
				icd.Entry{Variant: labeled},
				icd.Entry{Variant: line},
			)
			task = uuid.New()
			createICD(ctx, &ic, task)
		})

		It(
			"Should reject a reply with no identifier beside a streamed message",
			func(ctx SpecContext) {
				cfg := readConfig(
					bus.ReadMessage{Message: status.Key},
					bus.ReadMessage{Message: reading.Key},
				)
				Expect(resolver.Read(ctx, tx, task, &cfg)).To(MatchError(
					"messages.1.message: message Reading has a reply with no " +
						"identifier, so it cannot share the task with messages the " +
						"device sends unprompted: validation error",
				))
			},
		)

		It(
			"Should reject a text reply with no prefix beside a streamed message",
			func(ctx SpecContext) {
				cfg := readConfig(
					bus.ReadMessage{Message: line.Key},
					bus.ReadMessage{Message: status.Key},
				)
				Expect(resolver.Read(ctx, tx, task, &cfg)).To(MatchError(
					"messages.0.message: message Line has a reply with no " +
						"identifier, so it cannot share the task with messages the " +
						"device sends unprompted: validation error",
				))
			},
		)

		It(
			"Should accept replies with no identifier when no message is streamed",
			func(ctx SpecContext) {
				cfg := readConfig(
					bus.ReadMessage{Message: reading.Key},
					bus.ReadMessage{Message: line.Key},
					bus.ReadMessage{Message: labeled.Key},
				)
				Expect(resolver.Read(ctx, tx, task, &cfg)).To(Succeed())
			},
		)

		It(
			"Should accept a labeled reply beside a streamed message",
			func(ctx SpecContext) {
				cfg := readConfig(
					bus.ReadMessage{Message: status.Key},
					bus.ReadMessage{Message: labeled.Key},
				)
				Expect(resolver.Read(ctx, tx, task, &cfg)).To(Succeed())
			},
		)

		It(
			"Should ignore a disabled streamed message",
			func(ctx SpecContext) {
				cfg := readConfig(
					bus.ReadMessage{Message: status.Key, Disabled: true},
					bus.ReadMessage{Message: reading.Key},
				)
				Expect(resolver.Read(ctx, tx, task, &cfg)).To(Succeed())
			},
		)
	})

	Describe("Write", func() {
		It(
			"Should stamp the hash of the messages the config sends",
			func(ctx SpecContext) {
				cfg := bus.WriteConfig{
					ICD: &ic.Key,
					Messages: []bus.WriteMessage{{
						Message: command.Key,
						Fields: []bus.WriteField{
							{Field: throttle.Key, Channel: 3},
						},
					}},
				}
				Expect(resolver.Write(ctx, tx, task, &cfg)).To(Succeed())
				Expect(cfg.ICDHash).To(Equal(MustSucceed(
					icd.TaskHash(ic, []icd.EntryKey{command.Key}),
				)))
			},
		)

		It("Should reject a field that is not in the message", func(ctx SpecContext) {
			missing := uuid.New()
			cfg := bus.WriteConfig{
				ICD: &ic.Key,
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

		It(
			"Should accept a config with no ICD and no messages",
			func(ctx SpecContext) {
				cfg := bus.WriteConfig{}
				Expect(resolver.Write(ctx, tx, task, &cfg)).To(Succeed())
				Expect(cfg.ICDHash).To(BeEmpty())
			},
		)

		It(
			"Should reject a config with messages but no ICD",
			func(ctx SpecContext) {
				cfg := bus.WriteConfig{
					Messages: []bus.WriteMessage{{Message: command.Key}},
				}
				Expect(
					resolver.Write(ctx, tx, task, &cfg),
				).To(MatchError(ContainSubstring(
					"icd: select an ICD before choosing messages",
				)))
			},
		)
	})
})
