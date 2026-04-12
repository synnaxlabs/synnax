// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"context"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

type int8Entry struct {
	ID   int8
	Data string
}

func (e int8Entry) GorpKey() int8   { return e.ID }
func (int8Entry) SetOptions() []any { return nil }

type int16Entry struct {
	ID   int16
	Data string
}

func (e int16Entry) GorpKey() int16  { return e.ID }
func (int16Entry) SetOptions() []any { return nil }

type int64Entry struct {
	ID   int64
	Data string
}

func (e int64Entry) GorpKey() int64  { return e.ID }
func (int64Entry) SetOptions() []any { return nil }

type uint8Entry struct {
	ID   uint8
	Data string
}

func (e uint8Entry) GorpKey() uint8  { return e.ID }
func (uint8Entry) SetOptions() []any { return nil }

type uint16Entry struct {
	ID   uint16
	Data string
}

func (e uint16Entry) GorpKey() uint16 { return e.ID }
func (uint16Entry) SetOptions() []any { return nil }

type uint32Entry struct {
	ID   uint32
	Data string
}

func (e uint32Entry) GorpKey() uint32 { return e.ID }
func (uint32Entry) SetOptions() []any { return nil }

type uint64Entry struct {
	ID   uint64
	Data string
}

func (e uint64Entry) GorpKey() uint64 { return e.ID }
func (uint64Entry) SetOptions() []any { return nil }

type stringEntry struct {
	ID   string
	Data string
}

func (e stringEntry) GorpKey() string { return e.ID }
func (stringEntry) SetOptions() []any { return nil }

type namedStringKey string

type namedStringEntry struct {
	ID   namedStringKey
	Data string
}

func (e namedStringEntry) GorpKey() namedStringKey { return e.ID }
func (namedStringEntry) SetOptions() []any         { return nil }

var _ = Describe("KeyCodec", func() {
	var tx gorp.Tx
	BeforeEach(func() {
		tx = db.OpenTx()
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})

	Describe("Encode/Decode Roundtrip", func() {
		Describe("int8 keys", func() {
			DescribeTable("Should roundtrip",
				func(ctx SpecContext, id int8) {
					e := int8Entry{ID: id, Data: "data"}
					Expect(gorp.NewCreate[int8, int8Entry]().
						Entry(&e).Exec(ctx, tx)).To(Succeed())
					var res int8Entry
					Expect(gorp.NewRetrieve[int8, int8Entry]().
						WhereKeys(id).Entry(&res).Exec(ctx, tx)).To(Succeed())
					Expect(res).To(Equal(e))
				},
				Entry("zero", int8(0)),
				Entry("positive", int8(42)),
				Entry("negative", int8(-1)),
				Entry("max", int8(math.MaxInt8)),
				Entry("min", int8(math.MinInt8)),
			)
		})

		Describe("int16 keys", func() {
			DescribeTable("Should roundtrip",
				func(ctx SpecContext, id int16) {
					e := int16Entry{ID: id, Data: "data"}
					Expect(gorp.NewCreate[int16, int16Entry]().
						Entry(&e).Exec(ctx, tx)).To(Succeed())
					var res int16Entry
					Expect(gorp.NewRetrieve[int16, int16Entry]().
						WhereKeys(id).Entry(&res).Exec(ctx, tx)).To(Succeed())
					Expect(res).To(Equal(e))
				},
				Entry("zero", int16(0)),
				Entry("positive", int16(1000)),
				Entry("negative", int16(-1)),
				Entry("max", int16(math.MaxInt16)),
				Entry("min", int16(math.MinInt16)),
			)
		})

		Describe("int32 keys", func() {
			DescribeTable("Should roundtrip",
				func(ctx SpecContext, id int32) {
					e := entry{ID: id, Data: "data"}
					Expect(gorp.NewCreate[int32, entry]().
						Entry(&e).Exec(ctx, tx)).To(Succeed())
					var res entry
					Expect(gorp.NewRetrieve[int32, entry]().
						WhereKeys(id).Entry(&res).Exec(ctx, tx)).To(Succeed())
					Expect(res).To(Equal(e))
				},
				Entry("zero", int32(0)),
				Entry("positive", int32(100000)),
				Entry("negative", int32(-1)),
				Entry("max", int32(math.MaxInt32)),
				Entry("min", int32(math.MinInt32)),
			)
		})

		Describe("int64 keys", func() {
			DescribeTable("Should roundtrip",
				func(ctx SpecContext, id int64) {
					e := int64Entry{ID: id, Data: "data"}
					Expect(gorp.NewCreate[int64, int64Entry]().
						Entry(&e).Exec(ctx, tx)).To(Succeed())
					var res int64Entry
					Expect(gorp.NewRetrieve[int64, int64Entry]().
						WhereKeys(id).Entry(&res).Exec(ctx, tx)).To(Succeed())
					Expect(res).To(Equal(e))
				},
				Entry("zero", int64(0)),
				Entry("positive", int64(1e15)),
				Entry("negative", int64(-1)),
				Entry("max", int64(math.MaxInt64)),
				Entry("min", int64(math.MinInt64)),
			)
		})

		Describe("uint8 keys", func() {
			DescribeTable("Should roundtrip",
				func(ctx SpecContext, id uint8) {
					e := uint8Entry{ID: id, Data: "data"}
					Expect(gorp.NewCreate[uint8, uint8Entry]().
						Entry(&e).Exec(ctx, tx)).To(Succeed())
					var res uint8Entry
					Expect(gorp.NewRetrieve[uint8, uint8Entry]().
						WhereKeys(id).Entry(&res).Exec(ctx, tx)).To(Succeed())
					Expect(res).To(Equal(e))
				},
				Entry("zero", uint8(0)),
				Entry("positive", uint8(42)),
				Entry("max", uint8(math.MaxUint8)),
			)
		})

		Describe("uint16 keys", func() {
			DescribeTable("Should roundtrip",
				func(ctx SpecContext, id uint16) {
					e := uint16Entry{ID: id, Data: "data"}
					Expect(gorp.NewCreate[uint16, uint16Entry]().
						Entry(&e).Exec(ctx, tx)).To(Succeed())
					var res uint16Entry
					Expect(gorp.NewRetrieve[uint16, uint16Entry]().
						WhereKeys(id).Entry(&res).Exec(ctx, tx)).To(Succeed())
					Expect(res).To(Equal(e))
				},
				Entry("zero", uint16(0)),
				Entry("positive", uint16(1000)),
				Entry("max", uint16(math.MaxUint16)),
			)
		})

		Describe("uint32 keys", func() {
			DescribeTable("Should roundtrip",
				func(ctx SpecContext, id uint32) {
					e := uint32Entry{ID: id, Data: "data"}
					Expect(gorp.NewCreate[uint32, uint32Entry]().
						Entry(&e).Exec(ctx, tx)).To(Succeed())
					var res uint32Entry
					Expect(gorp.NewRetrieve[uint32, uint32Entry]().
						WhereKeys(id).Entry(&res).Exec(ctx, tx)).To(Succeed())
					Expect(res).To(Equal(e))
				},
				Entry("zero", uint32(0)),
				Entry("positive", uint32(100000)),
				Entry("max", uint32(math.MaxUint32)),
			)
		})

		Describe("uint64 keys", func() {
			DescribeTable("Should roundtrip",
				func(ctx SpecContext, id uint64) {
					e := uint64Entry{ID: id, Data: "data"}
					Expect(gorp.NewCreate[uint64, uint64Entry]().
						Entry(&e).Exec(ctx, tx)).To(Succeed())
					var res uint64Entry
					Expect(gorp.NewRetrieve[uint64, uint64Entry]().
						WhereKeys(id).Entry(&res).Exec(ctx, tx)).To(Succeed())
					Expect(res).To(Equal(e))
				},
				Entry("zero", uint64(0)),
				Entry("positive", uint64(1e18)),
				Entry("max", uint64(math.MaxUint64)),
			)
		})

		Describe("string keys", func() {
			DescribeTable("Should roundtrip",
				func(ctx SpecContext, id string) {
					e := stringEntry{ID: id, Data: "data"}
					Expect(gorp.NewCreate[string, stringEntry]().
						Entry(&e).Exec(ctx, tx)).To(Succeed())
					var res stringEntry
					Expect(gorp.NewRetrieve[string, stringEntry]().
						WhereKeys(id).Entry(&res).Exec(ctx, tx)).To(Succeed())
					Expect(res).To(Equal(e))
				},
				Entry("empty", ""),
				Entry("simple", "hello"),
				Entry("with special chars", "foo/bar:baz"),
			)
		})

		Describe("[]byte keys", func() {
			It("Should roundtrip", func(ctx SpecContext) {
				e := prefixEntry{ID: 42, Data: "data"}
				Expect(gorp.NewCreate[[]byte, prefixEntry]().
					Entry(&e).Exec(ctx, tx)).To(Succeed())
				var res prefixEntry
				Expect(gorp.NewRetrieve[[]byte, prefixEntry]().
					WhereKeys(e.GorpKey()).Entry(&res).Exec(ctx, tx)).To(Succeed())
				Expect(res).To(Equal(e))
			})
		})
	})

	Describe("Multiple entries with different key types", func() {
		It("Should create, retrieve, and delete uint16 entries", func(ctx SpecContext) {
			entries := []uint16Entry{
				{ID: 1, Data: "one"},
				{ID: 100, Data: "hundred"},
				{ID: math.MaxUint16, Data: "max"},
			}
			Expect(gorp.NewCreate[uint16, uint16Entry]().
				Entries(&entries).Exec(ctx, tx)).To(Succeed())
			var res []uint16Entry
			Expect(gorp.NewRetrieve[uint16, uint16Entry]().
				WhereKeys(1, 100, math.MaxUint16).
				Entries(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(3))
			Expect(gorp.NewDelete[uint16, uint16Entry]().
				WhereKeys(100).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewRetrieve[uint16, uint16Entry]().
				WhereKeys(100).Exists(ctx, tx)).To(BeFalse())
		})

		It("Should create, retrieve, and delete int64 entries", func(ctx SpecContext) {
			entries := []int64Entry{
				{ID: -1, Data: "neg"},
				{ID: 0, Data: "zero"},
				{ID: math.MaxInt64, Data: "max"},
			}
			Expect(gorp.NewCreate[int64, int64Entry]().
				Entries(&entries).Exec(ctx, tx)).To(Succeed())
			var res []int64Entry
			Expect(gorp.NewRetrieve[int64, int64Entry]().
				WhereKeys(-1, 0, math.MaxInt64).
				Entries(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(3))
		})
	})

	Describe("Key ordering", func() {
		It("Should iterate uint32 entries in big-endian byte order", func(ctx SpecContext) {
			entries := []uint32Entry{
				{ID: 300, Data: "three"},
				{ID: 1, Data: "one"},
				{ID: 200, Data: "two"},
			}
			Expect(gorp.NewCreate[uint32, uint32Entry]().
				Entries(&entries).Exec(ctx, tx)).To(Succeed())

			reader := gorp.WrapReader[uint32, uint32Entry](tx)
			iter := MustSucceed(reader.OpenIterator(gorp.IterOptions{}))
			defer func() { Expect(iter.Close()).To(Succeed()) }()

			var ids []uint32
			for iter.First(); iter.Valid(); iter.Next() {
				ids = append(ids, iter.Value(ctx).GorpKey())
			}
			Expect(iter.Error()).ToNot(HaveOccurred())
			Expect(ids).To(Equal([]uint32{1, 200, 300}))
		})

		It("Should iterate int16 entries in big-endian byte order", func(ctx SpecContext) {
			entries := []int16Entry{
				{ID: 300, Data: "three"},
				{ID: -1, Data: "neg"},
				{ID: 1, Data: "one"},
			}
			Expect(gorp.NewCreate[int16, int16Entry]().
				Entries(&entries).Exec(ctx, tx)).To(Succeed())

			reader := gorp.WrapReader[int16, int16Entry](tx)
			iter := MustSucceed(reader.OpenIterator(gorp.IterOptions{}))
			defer func() { Expect(iter.Close()).To(Succeed()) }()

			var ids []int16
			for iter.First(); iter.Valid(); iter.Next() {
				ids = append(ids, iter.Value(ctx).GorpKey())
			}
			Expect(iter.Error()).ToNot(HaveOccurred())
			Expect(ids).To(HaveLen(3))
		})
	})

	Describe("Type isolation", func() {
		It("Should not mix entries of different types with the same key value", func(ctx SpecContext) {
			e1 := uint32Entry{ID: 42, Data: "uint32"}
			e2 := int64Entry{ID: 42, Data: "int64"}
			Expect(gorp.NewCreate[uint32, uint32Entry]().
				Entry(&e1).Exec(ctx, tx)).To(Succeed())
			Expect(gorp.NewCreate[int64, int64Entry]().
				Entry(&e2).Exec(ctx, tx)).To(Succeed())

			var res1 uint32Entry
			Expect(gorp.NewRetrieve[uint32, uint32Entry]().
				WhereKeys(42).Entry(&res1).Exec(ctx, tx)).To(Succeed())
			Expect(res1.Data).To(Equal("uint32"))

			var res2 int64Entry
			Expect(gorp.NewRetrieve[int64, int64Entry]().
				WhereKeys(42).Entry(&res2).Exec(ctx, tx)).To(Succeed())
			Expect(res2.Data).To(Equal("int64"))
		})
	})

	Describe("Observe with non-int32 keys", func() {
		It("Should decode uint64 keys on delete notifications", func(ctx SpecContext) {
			uint64Table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[uint64, uint64Entry]{DB: db}))
			defer func() { Expect(uint64Table.Close()).To(Succeed()) }()

			Expect(gorp.NewCreate[uint64, uint64Entry]().
				Entry(&uint64Entry{ID: math.MaxUint64, Data: "data"}).
				Exec(ctx, db)).To(Succeed())

			tx := db.OpenTx()
			defer func() { Expect(tx.Close()).To(Succeed()) }()
			Expect(gorp.NewDelete[uint64, uint64Entry]().
				WhereKeys(math.MaxUint64).Exec(ctx, tx)).To(Succeed())

			var deletedKey uint64
			uint64Table.Observe().OnChange(
				func(_ context.Context, r gorp.TxReader[uint64, uint64Entry]) {
					for ch := range r {
						deletedKey = ch.Key
					}
				},
			)

			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(deletedKey).To(Equal(uint64(math.MaxUint64)))
		})

		It("Should decode int16 keys on set notifications", func(ctx SpecContext) {
			int16Table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[int16, int16Entry]{DB: db}))
			defer func() { Expect(int16Table.Close()).To(Succeed()) }()

			tx := db.OpenTx()
			defer func() { Expect(tx.Close()).To(Succeed()) }()
			Expect(gorp.NewCreate[int16, int16Entry]().
				Entry(&int16Entry{ID: -500, Data: "data"}).
				Exec(ctx, tx)).To(Succeed())

			var setKey int16
			int16Table.Observe().OnChange(
				func(_ context.Context, r gorp.TxReader[int16, int16Entry]) {
					for ch := range r {
						setKey = ch.Key
					}
				},
			)

			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(setKey).To(Equal(int16(-500)))
		})
	})

	Describe("Named string key types", func() {
		It("Should create, retrieve, and delete entries with a named string key", func(ctx SpecContext) {
			entries := []namedStringEntry{
				{ID: "alpha", Data: "first"},
				{ID: "beta", Data: "second"},
			}
			Expect(gorp.NewCreate[namedStringKey, namedStringEntry]().
				Entries(&entries).Exec(ctx, tx)).To(Succeed())

			var res []namedStringEntry
			Expect(gorp.NewRetrieve[namedStringKey, namedStringEntry]().
				WhereKeys(namedStringKey("alpha")).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(1))
			Expect(res[0].Data).To(Equal("first"))

			Expect(gorp.NewDelete[namedStringKey, namedStringEntry]().
				WhereKeys(namedStringKey("alpha")).
				Exec(ctx, tx)).To(Succeed())

			var res2 []namedStringEntry
			Expect(gorp.NewRetrieve[namedStringKey, namedStringEntry]().
				Entries(&res2).
				Exec(ctx, tx)).To(Succeed())
			Expect(res2).To(HaveLen(1))
			Expect(res2[0].ID).To(Equal(namedStringKey("beta")))
		})
	})

	Describe("WherePrefix with numeric keys", func() {
		It("Should retrieve entries matching a byte prefix for uint32 keys", func(ctx SpecContext) {
			entries := []uint32Entry{
				{ID: 0x01000000, Data: "a"},
				{ID: 0x01000001, Data: "b"},
				{ID: 0x02000000, Data: "c"},
			}
			Expect(gorp.NewCreate[uint32, uint32Entry]().
				Entries(&entries).Exec(ctx, tx)).To(Succeed())

			var res []uint32Entry
			Expect(gorp.NewRetrieve[uint32, uint32Entry]().
				WherePrefix([]byte{0x01}).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res).To(HaveLen(2))
		})
	})
})
