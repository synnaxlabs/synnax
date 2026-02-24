// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/gorp/testutil"
)

type entryV1 struct {
	ID   int32  `msgpack:"id"`
	Data string `msgpack:"data"`
}

func (e entryV1) GorpKey() int32    { return e.ID }
func (e entryV1) SetOptions() []any { return nil }

type entryV2 struct {
	ID          int32  `msgpack:"id"`
	Data        string `msgpack:"data"`
	Description string `msgpack:"description"`
}

func (e entryV2) GorpKey() int32    { return e.ID }
func (e entryV2) SetOptions() []any { return nil }

type jsonCodec struct{}

func (jsonCodec) Marshal(_ context.Context, e entryV2) ([]byte, error) {
	return json.Marshal(e)
}

func (jsonCodec) Unmarshal(_ context.Context, data []byte) (entryV2, error) {
	var e entryV2
	err := json.Unmarshal(data, &e)
	return e, err
}

var _ gorp.Codec[entryV2] = jsonCodec{}

var _ = Describe("Migrate Test Helpers", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("SeedAndMigrate", func() {
		It("Should seed and run a typed migration", func() {
			result := testutil.SeedAndMigrate[int32, entryV1, entryV2](
				ctx,
				[]entryV1{
					{ID: 1, Data: "one"},
					{ID: 2, Data: "two"},
				},
				nil,
				[]gorp.Migration{
					gorp.NewTypedMigration[entryV1, entryV2](
						"v1_to_v2",
						nil, nil,
						func(_ context.Context, old entryV1) (entryV2, error) {
							return entryV2{
								ID:          old.ID,
								Data:        old.Data,
								Description: "migrated:" + old.Data,
							}, nil
						},
						nil,
					),
				},
				nil,
			)
			defer func() { Expect(result.Close()).To(Succeed()) }()

			Expect(result.EntryCount(ctx)).To(Equal(2))
			e1 := result.Entry(ctx, int32(1))
			Expect(e1.Data).To(Equal("one"))
			Expect(e1.Description).To(Equal("migrated:one"))

			e2 := result.Entry(ctx, int32(2))
			Expect(e2.Data).To(Equal("two"))
			Expect(e2.Description).To(Equal("migrated:two"))
		})

		It("Should seed and run a codec transition migration", func() {
			codec := jsonCodec{}
			result := testutil.SeedAndMigrate[int32, entryV2, entryV2](
				ctx,
				[]entryV2{
					{ID: 1, Data: "one", Description: "first"},
				},
				nil,
				[]gorp.Migration{
					gorp.NewCodecTransition[int32, entryV2]("to_json", codec),
				},
				codec,
			)
			defer func() { Expect(result.Close()).To(Succeed()) }()

			e := result.Entry(ctx, int32(1))
			Expect(e.Data).To(Equal("one"))
			Expect(e.Description).To(Equal("first"))
		})

		It("Should seed with a custom codec and use matching inputCodec", func() {
			seedCodec := jsonCodec{}
			result := testutil.SeedAndMigrate[int32, entryV2, entryV2](
				ctx,
				[]entryV2{
					{ID: 1, Data: "json_seed", Description: "from_json"},
				},
				seedCodec,
				[]gorp.Migration{
					gorp.NewTypedMigration[entryV2, entryV2](
						"json_to_msgpack",
						jsonCodec{}, nil,
						func(_ context.Context, old entryV2) (entryV2, error) {
							return old, nil
						},
						nil,
					),
				},
				nil,
			)
			defer func() { Expect(result.Close()).To(Succeed()) }()

			e := result.Entry(ctx, int32(1))
			Expect(e.Data).To(Equal("json_seed"))
			Expect(e.Description).To(Equal("from_json"))
		})

		It("Should handle an empty seed", func() {
			result := testutil.SeedAndMigrate[int32, entryV1, entryV1](
				ctx,
				nil,
				nil,
				[]gorp.Migration{
					gorp.NewTypedMigration[entryV1, entryV1](
						"noop", nil, nil,
						func(_ context.Context, old entryV1) (entryV1, error) {
							return old, nil
						},
						nil,
					),
				},
				nil,
			)
			defer func() { Expect(result.Close()).To(Succeed()) }()
			Expect(result.EntryCount(ctx)).To(Equal(0))
		})
	})

	Describe("Migration chain", func() {
		It("Should run a chain of typed + codec transitions", func() {
			codec := jsonCodec{}
			result := testutil.SeedAndMigrate[int32, entryV1, entryV2](
				ctx,
				[]entryV1{
					{ID: 1, Data: "chain"},
				},
				nil,
				[]gorp.Migration{
					gorp.NewTypedMigration[entryV1, entryV2](
						"v1_to_v2",
						nil, nil,
						func(_ context.Context, old entryV1) (entryV2, error) {
							return entryV2{
								ID:          old.ID,
								Data:        old.Data,
								Description: "added_in_v2",
							}, nil
						},
						nil,
					),
					gorp.NewCodecTransition[int32, entryV2]("to_json", codec),
				},
				codec,
			)
			defer func() { Expect(result.Close()).To(Succeed()) }()

			e := result.Entry(ctx, int32(1))
			Expect(e.Data).To(Equal("chain"))
			Expect(e.Description).To(Equal("added_in_v2"))
		})
	})

	Describe("Version tracking", func() {
		It("Should report the correct migration version", func() {
			result := testutil.SeedAndMigrate[int32, entryV1, entryV1](
				ctx,
				[]entryV1{{ID: 1, Data: "versioned"}},
				nil,
				[]gorp.Migration{
					gorp.NewTypedMigration[entryV1, entryV1](
						"m1", nil, nil,
						func(_ context.Context, old entryV1) (entryV1, error) {
							return old, nil
						},
						nil,
					),
					gorp.NewTypedMigration[entryV1, entryV1](
						"m2", nil, nil,
						func(_ context.Context, old entryV1) (entryV1, error) {
							return old, nil
						},
						nil,
					),
				},
				nil,
			)
			defer func() { Expect(result.Close()).To(Succeed()) }()
			Expect(result.Version(ctx)).To(Equal(uint16(2)))
		})
	})

	Describe("Entries", func() {
		It("Should return all entries", func() {
			result := testutil.SeedAndMigrate[int32, entryV1, entryV1](
				ctx,
				[]entryV1{
					{ID: 1, Data: "a"},
					{ID: 2, Data: "b"},
					{ID: 3, Data: "c"},
				},
				nil,
				[]gorp.Migration{
					gorp.NewTypedMigration[entryV1, entryV1](
						"noop", nil, nil,
						func(_ context.Context, old entryV1) (entryV1, error) {
							return old, nil
						},
						nil,
					),
				},
				nil,
			)
			defer func() { Expect(result.Close()).To(Succeed()) }()
			entries := result.Entries(ctx)
			Expect(entries).To(HaveLen(3))
		})
	})

	Describe("RunAutoPost", func() {
		It("Should run auto only", func() {
			result := testutil.RunAutoPost(
				ctx,
				entryV1{ID: 1, Data: "input"},
				func(_ context.Context, old entryV1) (entryV2, error) {
					return entryV2{
						ID:          old.ID,
						Data:        old.Data,
						Description: "auto:" + old.Data,
					}, nil
				},
				nil,
			)
			Expect(result.ID).To(Equal(int32(1)))
			Expect(result.Data).To(Equal("input"))
			Expect(result.Description).To(Equal("auto:input"))
		})

		It("Should run auto then post", func() {
			result := testutil.RunAutoPost(
				ctx,
				entryV1{ID: 1, Data: "input"},
				func(_ context.Context, old entryV1) (entryV2, error) {
					return entryV2{
						ID:   old.ID,
						Data: old.Data,
					}, nil
				},
				func(_ context.Context, new *entryV2, old entryV1) error {
					new.Description = "post:" + old.Data
					return nil
				},
			)
			Expect(result.Description).To(Equal("post:input"))
		})

		It("Should run post only", func() {
			result := testutil.RunAutoPost[entryV1, entryV2](
				ctx,
				entryV1{ID: 1, Data: "input"},
				nil,
				func(_ context.Context, new *entryV2, old entryV1) error {
					new.ID = old.ID
					new.Data = old.Data
					new.Description = "post_only"
					return nil
				},
			)
			Expect(result.ID).To(Equal(int32(1)))
			Expect(result.Description).To(Equal("post_only"))
		})
	})
})
