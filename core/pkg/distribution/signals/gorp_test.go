// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type testUUIDEntry struct {
	Name string    `json:"name"`
	Key  uuid.UUID `json:"key"`
}

func (t testUUIDEntry) GorpKey() uuid.UUID { return t.Key }

func (t testUUIDEntry) SetOptions() []any { return nil }

type testNumericEntry struct {
	Value string `json:"value"`
	Key   uint32 `json:"key"`
}

func (t testNumericEntry) GorpKey() uint32 { return t.Key }

func (t testNumericEntry) SetOptions() []any { return nil }

type testStringEntry struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

func (t testStringEntry) GorpKey() string { return t.Key }

func (t testStringEntry) SetOptions() []any { return nil }

var _ = Describe("GorpPublisherConfig", func() {
	var (
		db          *gorp.DB
		uuidTable   *gorp.Table[uuid.UUID, testUUIDEntry]
		numTable    *gorp.Table[uint32, testNumericEntry]
		stringTable *gorp.Table[string, testStringEntry]
	)

	BeforeEach(func(ctx SpecContext) {
		db = gorp.Wrap(memkv.New())
		uuidTable = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[testUUIDEntry]{DB: db}))
		numTable = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[testNumericEntry]{DB: db}))
		stringTable = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[testStringEntry]{DB: db}))
	})

	AfterEach(func() {
		Expect(uuidTable.Close()).To(Succeed())
		Expect(numTable.Close()).To(Succeed())
		Expect(stringTable.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})

	Describe("MarshalJSON", func() {
		It("Should marshal an entry to JSON with uint32 length prefix", func() {
			entry := testUUIDEntry{
				Key:  uuid.MustParse("12345678-1234-1234-1234-123456789012"),
				Name: "test",
			}
			b := MustSucceed(signals.MarshalJSON(entry))
			Expect(len(b)).To(BeNumerically(">=", 4))
			s := telem.Series{DataType: telem.JSONT, Data: b}
			Expect(s.Len()).To(Equal(int64(1)))
		})
	})

	Describe("GorpPublisherConfigUUID", func() {
		It("Should create a config for UUID keyed entries", func() {
			cfg := signals.GorpPublisherConfigUUID[testUUIDEntry](uuidTable.Observe())
			Expect(cfg.Observable).ToNot(BeNil())
			Expect(cfg.DeleteDataType).To(Equal(telem.UUIDT))
			Expect(cfg.SetDataType).To(Equal(telem.JSONT))
			Expect(cfg.MarshalDelete).ToNot(BeNil())
			Expect(cfg.MarshalSet).ToNot(BeNil())
		})

		It("Should correctly marshal UUID for delete", func() {
			cfg := signals.GorpPublisherConfigUUID[testUUIDEntry](uuidTable.Observe())
			uid := uuid.MustParse("12345678-1234-1234-1234-123456789012")
			b := MustSucceed(cfg.MarshalDelete(uid))
			Expect(b).To(Equal(uid[:]))
		})

		It("Should correctly marshal entry for set", func() {
			cfg := signals.GorpPublisherConfigUUID[testUUIDEntry](uuidTable.Observe())
			entry := testUUIDEntry{
				Key:  uuid.MustParse("12345678-1234-1234-1234-123456789012"),
				Name: "test-entry",
			}
			b := MustSucceed(cfg.MarshalSet(entry))
			Expect(string(b)).To(ContainSubstring(`"name":"test-entry"`))
		})
	})

	Describe("GorpPublisherConfigNumeric", func() {
		It("Should create a config for numeric keyed entries with JSON set", func() {
			cfg := signals.GorpPublisherConfigNumeric[uint32, testNumericEntry](numTable.Observe(), telem.Uint32T)
			Expect(cfg.Observable).ToNot(BeNil())
			Expect(cfg.DeleteDataType).To(Equal(telem.Uint32T))
			Expect(cfg.SetDataType).To(Equal(telem.JSONT))
			Expect(cfg.MarshalDelete).ToNot(BeNil())
			Expect(cfg.MarshalSet).ToNot(BeNil())
		})

		It("Should correctly marshal numeric key for delete", func() {
			cfg := signals.GorpPublisherConfigNumeric[uint32, testNumericEntry](numTable.Observe(), telem.Uint32T)
			b := MustSucceed(cfg.MarshalDelete(42))
			Expect(b).To(HaveLen(4)) // uint32 is 4 bytes
		})

		It("Should correctly marshal entry for set as JSON", func() {
			cfg := signals.GorpPublisherConfigNumeric[uint32, testNumericEntry](numTable.Observe(), telem.Uint32T)
			entry := testNumericEntry{Key: 123, Value: "test-value"}
			b := MustSucceed(cfg.MarshalSet(entry))
			Expect(string(b)).To(ContainSubstring(`"value":"test-value"`))
		})
	})

	Describe("GorpPublisherConfigPureNumeric", func() {
		It("Should create a config for numeric keyed entries with numeric set", func() {
			cfg := signals.GorpPublisherConfigPureNumeric[uint32, testNumericEntry](numTable.Observe(), telem.Uint32T)
			Expect(cfg.Observable).ToNot(BeNil())
			Expect(cfg.DeleteDataType).To(Equal(telem.Uint32T))
			Expect(cfg.SetDataType).To(Equal(telem.Uint32T))
			Expect(cfg.MarshalDelete).ToNot(BeNil())
			Expect(cfg.MarshalSet).ToNot(BeNil())
		})

		It("Should correctly marshal numeric key for delete", func() {
			cfg := signals.GorpPublisherConfigPureNumeric[uint32, testNumericEntry](numTable.Observe(), telem.Uint32T)
			b := MustSucceed(cfg.MarshalDelete(42))
			Expect(b).To(HaveLen(4)) // uint32 is 4 bytes
		})

		It("Should correctly marshal entry key for set", func() {
			cfg := signals.GorpPublisherConfigPureNumeric[uint32, testNumericEntry](numTable.Observe(), telem.Uint32T)
			entry := testNumericEntry{Key: 999, Value: "ignored"}
			b := MustSucceed(cfg.MarshalSet(entry))
			Expect(b).To(HaveLen(4)) // uint32 is 4 bytes
		})
	})

	Describe("GorpPublisherConfigString", func() {
		It("Should create a config for string keyed entries", func() {
			cfg := signals.GorpPublisherConfigString[testStringEntry](stringTable.Observe())
			Expect(cfg.Observable).ToNot(BeNil())
			Expect(cfg.DeleteDataType).To(Equal(telem.StringT))
			Expect(cfg.SetDataType).To(Equal(telem.JSONT))
			Expect(cfg.MarshalDelete).ToNot(BeNil())
			Expect(cfg.MarshalSet).ToNot(BeNil())
		})

		It("Should correctly marshal string key for delete with length prefix", func() {
			cfg := signals.GorpPublisherConfigString[testStringEntry](stringTable.Observe())
			b := MustSucceed(cfg.MarshalDelete("my-key"))
			s := telem.Series{DataType: telem.StringT, Data: b}
			Expect(s.Len()).To(Equal(int64(1)))
			Expect(string(s.At(0))).To(Equal("my-key"))
		})

		It("Should correctly marshal entry for set as JSON", func() {
			cfg := signals.GorpPublisherConfigString[testStringEntry](stringTable.Observe())
			entry := testStringEntry{Key: "entry-key", Value: 42}
			b := MustSucceed(cfg.MarshalSet(entry))
			Expect(string(b)).To(ContainSubstring(`"value":42`))
		})
	})

	Describe("DefaultGorpPublisherConfig", func() {
		It("Should generate default channel names based on type name", func() {
			cfg := signals.DefaultGorpPublisherConfig[uuid.UUID, testUUIDEntry]()
			Expect(cfg.SetName).To(Equal("sy_testuuidentry_set"))
			Expect(cfg.DeleteName).To(Equal("sy_testuuidentry_delete"))
		})
	})

	Describe("GorpPublisherConfig validation", func() {
		It("Should validate required fields", func() {
			cfg := signals.GorpPublisherConfig[uuid.UUID, testUUIDEntry]{}
			err := cfg.Validate()
			Expect(err).To(HaveOccurred())
		})

		It("Should pass validation with all required fields", func() {
			cfg := signals.GorpPublisherConfigUUID[testUUIDEntry](uuidTable.Observe())
			cfg.SetName = "test_set"
			cfg.DeleteName = "test_delete"
			Expect(cfg.Validate()).To(Succeed())
		})
	})
})
