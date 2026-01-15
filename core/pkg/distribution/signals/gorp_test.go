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
)

type testUUIDEntry struct {
	Key  uuid.UUID `json:"key"`
	Name string    `json:"name"`
}

func (t testUUIDEntry) GorpKey() uuid.UUID { return t.Key }

func (t testUUIDEntry) SetOptions() []interface{} { return nil }

type testNumericEntry struct {
	Key   uint32 `json:"key"`
	Value string `json:"value"`
}

func (t testNumericEntry) GorpKey() uint32 { return t.Key }

func (t testNumericEntry) SetOptions() []interface{} { return nil }

type testStringEntry struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

func (t testStringEntry) GorpKey() string { return t.Key }

func (t testStringEntry) SetOptions() []interface{} { return nil }

var _ = Describe("GorpPublisherConfig", func() {
	var db *gorp.DB

	BeforeEach(func() {
		db = gorp.Wrap(memkv.New())
	})

	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})

	Describe("MarshalJSON", func() {
		It("Should marshal an entry to JSON with newline suffix", func() {
			entry := testUUIDEntry{
				Key:  uuid.MustParse("12345678-1234-1234-1234-123456789012"),
				Name: "test",
			}
			b, err := signals.MarshalJSON[uuid.UUID, testUUIDEntry](entry)
			Expect(err).ToNot(HaveOccurred())
			Expect(b).To(HaveSuffix("\n"))
			Expect(string(b)).To(ContainSubstring(`"name":"test"`))
		})
	})

	Describe("GorpPublisherConfigUUID", func() {
		It("Should create a config for UUID keyed entries", func() {
			cfg := signals.GorpPublisherConfigUUID[testUUIDEntry](db)
			Expect(cfg.DB).To(Equal(db))
			Expect(cfg.DeleteDataType).To(Equal(telem.UUIDT))
			Expect(cfg.SetDataType).To(Equal(telem.JSONT))
			Expect(cfg.MarshalDelete).ToNot(BeNil())
			Expect(cfg.MarshalSet).ToNot(BeNil())
		})

		It("Should correctly marshal UUID for delete", func() {
			cfg := signals.GorpPublisherConfigUUID[testUUIDEntry](db)
			uid := uuid.MustParse("12345678-1234-1234-1234-123456789012")
			b, err := cfg.MarshalDelete(uid)
			Expect(err).ToNot(HaveOccurred())
			Expect(b).To(Equal(uid[:]))
		})

		It("Should correctly marshal entry for set", func() {
			cfg := signals.GorpPublisherConfigUUID[testUUIDEntry](db)
			entry := testUUIDEntry{
				Key:  uuid.MustParse("12345678-1234-1234-1234-123456789012"),
				Name: "test-entry",
			}
			b, err := cfg.MarshalSet(entry)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(b)).To(ContainSubstring(`"name":"test-entry"`))
		})
	})

	Describe("GorpPublisherConfigNumeric", func() {
		It("Should create a config for numeric keyed entries with JSON set", func() {
			cfg := signals.GorpPublisherConfigNumeric[uint32, testNumericEntry](db, telem.Uint32T)
			Expect(cfg.DB).To(Equal(db))
			Expect(cfg.DeleteDataType).To(Equal(telem.Uint32T))
			Expect(cfg.SetDataType).To(Equal(telem.JSONT))
			Expect(cfg.MarshalDelete).ToNot(BeNil())
			Expect(cfg.MarshalSet).ToNot(BeNil())
		})

		It("Should correctly marshal numeric key for delete", func() {
			cfg := signals.GorpPublisherConfigNumeric[uint32, testNumericEntry](db, telem.Uint32T)
			b, err := cfg.MarshalDelete(42)
			Expect(err).ToNot(HaveOccurred())
			Expect(b).To(HaveLen(4)) // uint32 is 4 bytes
		})

		It("Should correctly marshal entry for set as JSON", func() {
			cfg := signals.GorpPublisherConfigNumeric[uint32, testNumericEntry](db, telem.Uint32T)
			entry := testNumericEntry{Key: 123, Value: "test-value"}
			b, err := cfg.MarshalSet(entry)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(b)).To(ContainSubstring(`"value":"test-value"`))
		})
	})

	Describe("GorpPublisherConfigPureNumeric", func() {
		It("Should create a config for numeric keyed entries with numeric set", func() {
			cfg := signals.GorpPublisherConfigPureNumeric[uint32, testNumericEntry](db, telem.Uint32T)
			Expect(cfg.DB).To(Equal(db))
			Expect(cfg.DeleteDataType).To(Equal(telem.Uint32T))
			Expect(cfg.SetDataType).To(Equal(telem.Uint32T))
			Expect(cfg.MarshalDelete).ToNot(BeNil())
			Expect(cfg.MarshalSet).ToNot(BeNil())
		})

		It("Should correctly marshal numeric key for delete", func() {
			cfg := signals.GorpPublisherConfigPureNumeric[uint32, testNumericEntry](db, telem.Uint32T)
			b, err := cfg.MarshalDelete(42)
			Expect(err).ToNot(HaveOccurred())
			Expect(b).To(HaveLen(4)) // uint32 is 4 bytes
		})

		It("Should correctly marshal entry key for set", func() {
			cfg := signals.GorpPublisherConfigPureNumeric[uint32, testNumericEntry](db, telem.Uint32T)
			entry := testNumericEntry{Key: 999, Value: "ignored"}
			b, err := cfg.MarshalSet(entry)
			Expect(err).ToNot(HaveOccurred())
			Expect(b).To(HaveLen(4)) // uint32 is 4 bytes
		})
	})

	Describe("GorpPublisherConfigString", func() {
		It("Should create a config for string keyed entries", func() {
			cfg := signals.GorpPublisherConfigString[testStringEntry](db)
			Expect(cfg.DB).To(Equal(db))
			Expect(cfg.DeleteDataType).To(Equal(telem.StringT))
			Expect(cfg.SetDataType).To(Equal(telem.JSONT))
			Expect(cfg.MarshalDelete).ToNot(BeNil())
			Expect(cfg.MarshalSet).ToNot(BeNil())
		})

		It("Should correctly marshal string key for delete with newline", func() {
			cfg := signals.GorpPublisherConfigString[testStringEntry](db)
			b, err := cfg.MarshalDelete("my-key")
			Expect(err).ToNot(HaveOccurred())
			Expect(string(b)).To(Equal("my-key\n"))
		})

		It("Should correctly marshal entry for set as JSON", func() {
			cfg := signals.GorpPublisherConfigString[testStringEntry](db)
			entry := testStringEntry{Key: "entry-key", Value: 42}
			b, err := cfg.MarshalSet(entry)
			Expect(err).ToNot(HaveOccurred())
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
			cfg := signals.GorpPublisherConfigUUID[testUUIDEntry](db)
			cfg.SetName = "test_set"
			cfg.DeleteName = "test_delete"
			err := cfg.Validate()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
