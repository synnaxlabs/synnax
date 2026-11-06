// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type wideRecord struct {
	Key   uuid.UUID `json:"key"   msgpack:"key"`
	Value int32     `json:"value" msgpack:"value"`
}

func (r wideRecord) GorpKey() uuid.UUID   { return r.Key }
func (wideRecord) SetOptions() []any      { return nil }
func (wideRecord) CustomTypeName() string { return "record_test_config" }

type narrowRecord struct {
	Key   uuid.UUID `json:"key"   msgpack:"key"`
	Value uint8     `json:"value" msgpack:"value"`
}

func (r narrowRecord) GorpKey() uuid.UUID   { return r.Key }
func (narrowRecord) SetOptions() []any      { return nil }
func (narrowRecord) CustomTypeName() string { return "record_test_config" }

// EncodeOrc writes the record at the wide fixed width.
func (r wideRecord) EncodeOrc(w *orc.Writer) error {
	w.Write(r.Key[:])
	w.Int32(r.Value)
	return nil
}

// DecodeOrc reads the record at the wide fixed width.
func (r *wideRecord) DecodeOrc(o *orc.Reader) (err error) {
	if _, err = o.Read(r.Key[:]); err != nil {
		return err
	}
	r.Value, err = o.Int32()
	return err
}

// EncodeOrc writes the record at the narrow fixed width.
func (r narrowRecord) EncodeOrc(w *orc.Writer) error {
	w.Write(r.Key[:])
	w.Uint8(r.Value)
	return nil
}

// DecodeOrc reads the record at the narrow fixed width.
func (r *narrowRecord) DecodeOrc(o *orc.Reader) (err error) {
	if _, err = o.Read(r.Key[:]); err != nil {
		return err
	}
	r.Value, err = o.Uint8()
	return err
}

func liftRecord(_ context.Context, r wideRecord) (narrowRecord, error) {
	if r.Value < 0 || r.Value > 255 {
		return narrowRecord{}, errors.New("value outside the range")
	}
	return narrowRecord{Key: r.Key, Value: uint8(r.Value)}, nil
}

var _ = Describe("NewRecordMigration", func() {
	var (
		db  *gorp.DB
		otg *ontology.Ontology
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	})

	store := func(ctx SpecContext, records ...wideRecord) {
		GinkgoHelper()
		Expect(gorp.NewCreate[uuid.UUID, wideRecord]().Entries(&records).Exec(ctx, db)).
			To(Succeed())
	}

	run := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "record_test_config",
			Migrations: []gorp.Migration{
				v3.NewRecordMigration("narrow_record", liftRecord, otg),
			},
		})).To(Succeed())
	}

	exists := func(ctx SpecContext, key uuid.UUID) bool {
		GinkgoHelper()
		return MustSucceed(gorp.NewRetrieve[uuid.UUID, narrowRecord]().
			Where(gorp.MatchKeys[uuid.UUID, narrowRecord](key)).
			Exists(ctx, db))
	}

	It("Should write each lifted record in the new shape", func(ctx SpecContext) {
		key := uuid.New()
		store(ctx, wideRecord{Key: key, Value: 7})
		run(ctx)
		var got narrowRecord
		Expect(gorp.NewRetrieve[uuid.UUID, narrowRecord]().
			Where(gorp.MatchKeys[uuid.UUID, narrowRecord](key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Value).To(Equal(uint8(7)))
	})

	It("Should quarantine a rejected record with its task", func(ctx SpecContext) {
		bad, good := uuid.New(), uuid.New()
		store(ctx, wideRecord{Key: bad, Value: 300}, wideRecord{Key: good, Value: 1})
		t := v2.Task{Key: bad, Name: "Stored Task", Type: "ni_digital_read"}
		Expect(gorp.NewCreate[v2.Key, v2.Task]().Entry(&t).Exec(ctx, db)).To(Succeed())
		run(ctx)
		Expect(exists(ctx, bad)).To(BeFalse())
		Expect(exists(ctx, good)).To(BeTrue())
		Expect(gorp.NewRetrieve[v2.Key, v2.Task]().
			Where(gorp.MatchKeys[v2.Key, v2.Task](bad)).
			Exists(ctx, db)).To(BeFalse())
		staged, closer := MustSucceed2(db.Get(ctx, v3.QuarantineKVKey(bad)))
		Expect(string(staged)).To(ContainSubstring("Stored Task"))
		Expect(string(staged)).To(ContainSubstring("300"))
		Expect(closer.Close()).To(Succeed())
	})

	It("Should quarantine a rejected record that has no task", func(ctx SpecContext) {
		key := uuid.New()
		store(ctx, wideRecord{Key: key, Value: -1})
		run(ctx)
		Expect(exists(ctx, key)).To(BeFalse())
		staged, closer := MustSucceed2(db.Get(ctx, v3.QuarantineKVKey(key)))
		Expect(string(staged)).To(ContainSubstring(key.String()))
		Expect(closer.Close()).To(Succeed())
	})

	It("Should leave no quarantine entry for a lifted record", func(ctx SpecContext) {
		key := uuid.New()
		store(ctx, wideRecord{Key: key, Value: 1})
		run(ctx)
		Expect(db.Get(ctx, v3.QuarantineKVKey(key))).Error().
			To(MatchError(query.ErrNotFound))
	})
})
