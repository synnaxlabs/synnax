// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"fmt"
	"sync/atomic"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", Ordered, func() {
	var (
		db    *gorp.DB
		svc   *status.Service
		racks *gorp.Table[rack.Key, rack.Rack]
	)
	// setStatus writes st in its own transaction, the way a caller outside an existing
	// operation does.
	setStatus := func(ctx SpecContext, st *status.Status[any]) error {
		GinkgoHelper()
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Set(ctx, st)
		})
	}
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		// Status labels reference the label table, so it must be open.
		MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: db,
		}))
		racks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[rack.Key, rack.Rack]{
			DB: db, Name: string(resource.TypeRack),
		}))
		svc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: db,
		}))
	})

	Describe("SetByKeyOrName", func() {
		Describe("Input validation", func() {
			It(
				"Should return a validation error for an unknown variant",
				func(ctx SpecContext) {
					Expect(svc.SetByKeyOrName(ctx, "dispatch_iv_a", "msg", "bogus")).
						Error().
						To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("invalid status variant"))))
				},
			)

			It(
				"Should reject upper-cased variants (case-sensitive)",
				func(ctx SpecContext) {
					Expect(svc.SetByKeyOrName(ctx, "dispatch_iv_b", "msg", "SUCCESS")).
						Error().
						To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("invalid status variant"))))
				},
			)

			It("Should reject the empty variant", func(ctx SpecContext) {
				Expect(svc.SetByKeyOrName(ctx, "dispatch_iv_c", "msg", "")).
					Error().
					To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("invalid status variant"))))
			})

			It(
				"Should return a validation error for empty input",
				func(ctx SpecContext) {
					Expect(
						svc.SetByKeyOrName(ctx, "", "msg", string(status.VariantInfo)),
					).
						Error().
						To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("key_or_name is required"))))
				},
			)

			It(
				"Should not write anything to the store when the variant is invalid",
				func(ctx SpecContext) {
					name := "dispatch_iv_unwritten"
					Expect(svc.SetByKeyOrName(ctx, name, "msg", "bogus")).
						Error().
						To(SatisfyAll(MatchError(validate.ErrValidation), MatchError(ContainSubstring("invalid status variant"))))
					Expect(svc.NewRetrieve[any]().Where(status.MatchKeys[any](name)).
						Entry(&status.Status[any]{}).
						Exec(ctx, db)).To(MatchError(query.ErrNotFound))
				},
			)
		})

		Describe("By-key path", func() {
			It(
				"Should update an existing row whose Key matches the input",
				func(ctx SpecContext) {
					key := uuid.New().String()
					Expect(setStatus(ctx, &status.Status[any]{
						Key: key, Name: "by_key_orig", Variant: status.VariantInfo,
						Message: "old", Time: telem.Now(),
					})).To(Succeed())

					gotKey, multi := MustSucceed2(
						svc.SetByKeyOrName(
							ctx,
							key,
							"new",
							string(status.VariantWarning),
						),
					)
					Expect(gotKey).To(Equal(key))
					Expect(multi).To(BeFalse())

					var s status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchKeys[any](key)).
							Entry(&s).
							Exec(ctx, db),
					).To(Succeed())
					Expect(s.Variant).To(Equal(status.VariantWarning))
					Expect(s.Message).To(Equal("new"))
					Expect(s.Time).ToNot(BeZero())
				},
			)

			It(
				"Should clear the description that explained the previous message",
				func(ctx SpecContext) {
					key := uuid.New().String()
					Expect(setStatus(ctx, &status.Status[any]{
						Key:         key,
						Name:        "by_key_described",
						Variant:     status.VariantError,
						Message:     "old",
						Description: "The old cause.",
						Time:        telem.Now(),
					})).To(Succeed())
					MustSucceed2(
						svc.SetByKeyOrName(ctx, key, "new", string(status.VariantInfo)),
					)
					var s status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchKeys[any](key)).
							Entry(&s).
							Exec(ctx, db),
					).To(Succeed())
					Expect(s.Message).To(Equal("new"))
					Expect(s.Description).To(BeEmpty())
				},
			)

			It("Should match arbitrary (non-UUID) string keys", func(ctx SpecContext) {
				key := "by_key_plain_string"
				Expect(setStatus(ctx, &status.Status[any]{
					Key: key, Name: "by_key_plain_orig", Variant: status.VariantInfo,
					Message: "old", Time: telem.Now(),
				})).To(Succeed())

				gotKey, multi := MustSucceed2(
					svc.SetByKeyOrName(ctx, key, "new", string(status.VariantSuccess)),
				)
				Expect(gotKey).To(Equal(key))
				Expect(multi).To(BeFalse())
			})

			It(
				"Should prefer the by-key match over a by-name match for the same input",
				func(ctx SpecContext) {
					shared := "shared_token"
					Expect(setStatus(ctx, &status.Status[any]{
						Key: shared, Name: "by_key_winner", Variant: status.VariantInfo,
						Message: "key", Time: telem.Now(),
					})).To(Succeed())
					Expect(setStatus(ctx, &status.Status[any]{
						Key:     uuid.New().String(),
						Name:    shared,
						Variant: status.VariantInfo,
						Message: "name",
						Time:    telem.Now(),
					})).To(Succeed())

					gotKey, multi := MustSucceed2(
						svc.SetByKeyOrName(
							ctx,
							shared,
							"updated",
							string(status.VariantWarning),
						),
					)
					Expect(gotKey).To(Equal(shared))
					Expect(multi).To(BeFalse())
				},
			)
		})

		Describe("By-name path", func() {
			It(
				"Should update in place when there is a single name match",
				func(ctx SpecContext) {
					name := "by_name_single"
					existingKey := uuid.New().String()
					Expect(setStatus(ctx, &status.Status[any]{
						Key: existingKey, Name: name, Variant: status.VariantSuccess,
						Message: "ok", Time: telem.Now(),
					})).To(Succeed())

					gotKey, multi := MustSucceed2(
						svc.SetByKeyOrName(
							ctx,
							name,
							"now bad",
							string(status.VariantError),
						),
					)
					Expect(gotKey).To(Equal(existingKey))
					Expect(multi).To(BeFalse())

					var s status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchKeys[any](existingKey)).
							Entry(&s).
							Exec(ctx, db),
					).To(Succeed())
					Expect(s.Variant).To(Equal(status.VariantError))
					Expect(s.Message).To(Equal("now bad"))
				},
			)

			It(
				"Should report multipleMatches and update only the first match",
				func(ctx SpecContext) {
					name := "by_name_multi"
					firstKey := uuid.New().String()
					secondKey := uuid.New().String()
					Expect(setStatus(ctx, &status.Status[any]{
						Key: firstKey, Name: name, Variant: status.VariantInfo,
						Message: "first", Time: telem.Now(),
					})).To(Succeed())
					Expect(setStatus(ctx, &status.Status[any]{
						Key: secondKey, Name: name, Variant: status.VariantInfo,
						Message: "second", Time: telem.Now(),
					})).To(Succeed())

					gotKey, multi := MustSucceed2(
						svc.SetByKeyOrName(
							ctx,
							name,
							"updated",
							string(status.VariantWarning),
						),
					)
					Expect(multi).To(BeTrue())
					Expect(gotKey).To(SatisfyAny(Equal(firstKey), Equal(secondKey)))

					var rows []status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchKeys[any](firstKey, secondKey)).
							Entries(&rows).
							Exec(ctx, db),
					).To(Succeed())
					Expect(rows).To(HaveLen(2))
					warning, info := 0, 0
					for _, r := range rows {
						switch r.Variant {
						case status.VariantWarning:
							warning++
						case status.VariantInfo:
							info++
						}
					}
					Expect(warning).To(Equal(1))
					Expect(info).To(Equal(1))
				},
			)

			It(
				"Should set the status with the lowest key each time a name repeats",
				func(ctx SpecContext) {
					name := "by_name_order"
					for i := range 16 {
						Expect(setStatus(ctx, &status.Status[any]{
							Key:     fmt.Sprintf("%s_%02d", name, i),
							Name:    name,
							Variant: status.VariantInfo,
							Time:    telem.Now(),
						})).To(Succeed())
					}
					for range 5 {
						gotKey, multi := MustSucceed2(svc.SetByKeyOrName(
							ctx,
							name,
							"updated",
							string(status.VariantWarning),
						))
						Expect(gotKey).To(Equal(name + "_00"))
						Expect(multi).To(BeTrue())
					}
				},
			)

			It(
				"Should accept an empty message on the by-name path",
				func(ctx SpecContext) {
					name := "by_name_empty_msg"
					existingKey := uuid.New().String()
					Expect(setStatus(ctx, &status.Status[any]{
						Key: existingKey, Name: name, Variant: status.VariantInfo,
						Message: "old", Time: telem.Now(),
					})).To(Succeed())
					gotKey, _ := MustSucceed2(
						svc.SetByKeyOrName(ctx, name, "", string(status.VariantInfo)),
					)
					Expect(gotKey).To(Equal(existingKey))
					var s status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchKeys[any](gotKey)).
							Entry(&s).
							Exec(ctx, db),
					).To(Succeed())
					Expect(s.Message).To(BeEmpty())
				},
			)
		})

		Describe("Create path", func() {
			It(
				"Should create a UUID-keyed row named after the input when nothing matches",
				func(ctx SpecContext) {
					input := "fresh_row_a"
					gotKey, multi := MustSucceed2(
						svc.SetByKeyOrName(
							ctx,
							input,
							"hello",
							string(status.VariantInfo),
						),
					)
					MustSucceed(uuid.Parse(gotKey))
					Expect(gotKey).ToNot(Equal(input))
					Expect(multi).To(BeFalse())

					var s status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchKeys[any](gotKey)).
							Entry(&s).
							Exec(ctx, db),
					).To(Succeed())
					Expect(s.Key).To(Equal(gotKey))
					Expect(s.Name).To(Equal(input))
					Expect(s.Variant).To(Equal(status.VariantInfo))
					Expect(s.Message).To(Equal("hello"))
				},
			)

			It(
				"Should match a created row by name on subsequent SetByKeyOrName calls",
				func(ctx SpecContext) {
					input := "fresh_row_then_update"
					firstKey, _ := MustSucceed2(
						svc.SetByKeyOrName(
							ctx,
							input,
							"first",
							string(status.VariantInfo),
						),
					)
					gotKey, multi := MustSucceed2(
						svc.SetByKeyOrName(
							ctx,
							input,
							"second",
							string(status.VariantWarning),
						),
					)
					Expect(gotKey).To(Equal(firstKey))
					Expect(multi).To(BeFalse())

					var s status.Status[any]
					Expect(
						svc.NewRetrieve[any]().
							Where(status.MatchKeys[any](gotKey)).
							Entry(&s).
							Exec(ctx, db),
					).To(Succeed())
					Expect(s.Message).To(Equal("second"))
					Expect(s.Variant).To(Equal(status.VariantWarning))
				},
			)
		})
	})

	Describe("Join", func() {
		It("Should resolve and join statuses through a snapshot", func(
			ctx SpecContext,
		) {
			r := rack.Rack{Key: 65539, Name: "snapshot"}
			Expect(racks.NewCreate().Entry(&r).Exec(ctx, db)).To(Succeed())
			owner := r.ResourceID()
			Expect(setStatus(ctx, &status.Status[any]{
				Owner:   &owner,
				Name:    "snapshot-join",
				Variant: status.VariantSuccess,
				Message: "running",
				Time:    telem.Now(),
			})).To(Succeed())
			var (
				resolved []status.Status[any]
				joined   []*status.Status[any]
			)
			Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
				if resolved, err = svc.ResolveKeyOrName(
					ctx,
					snap,
					"snapshot-join",
				); err != nil {
					return err
				}
				joined, err = svc.Join[any](ctx, snap, []resource.ID{owner})
				return err
			})).To(Succeed())
			Expect(resolved).To(HaveExactElements(HaveField("Name", "snapshot-join")))
			Expect(joined).To(HaveExactElements(HaveField("Message", "running")))
		})

		It(
			"Should return the status of each resource in order, nil for a missing one",
			func(ctx SpecContext) {
				r := rack.Rack{Key: 65537, Name: "stored"}
				Expect(racks.NewCreate().Entry(&r).Exec(ctx, db)).To(Succeed())
				stored := r.ResourceID()
				missing := rack.Key(65538).ResourceID()
				Expect(setStatus(ctx, &status.Status[any]{
					Owner:   &stored,
					Name:    "stored",
					Variant: status.VariantSuccess,
					Message: "running",
					Time:    telem.Now(),
				})).To(Succeed())
				joined := MustSucceed(
					svc.Join[any](ctx, db, []resource.ID{missing, stored}),
				)
				Expect(joined).To(HaveLen(2))
				Expect(joined[0]).To(BeNil())
				Expect(joined[1].Owner).To(HaveValue(Equal(stored)))
				Expect(joined[1].Message).To(Equal("running"))
			},
		)
	})
})

// failOnceDB fails the first iterator that the DB opens, which is the scan that
// populates the indexes of a table.
type failOnceDB struct {
	kv.DB
	failed atomic.Bool
}

func (f *failOnceDB) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	if f.failed.CompareAndSwap(false, true) {
		return nil, errors.New("populate failed")
	}
	return f.DB.OpenIterator(opts)
}

var _ = Describe("Failed index population", func() {
	It("Should find a status by name with a scan", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(&failOnceDB{DB: memkv.New()}))
		svc := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		st := status.Status[any]{
			Key:     "stored",
			Name:    "pump",
			Variant: status.VariantInfo,
			Time:    telem.Now(),
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Set(ctx, &st)
		})).To(Succeed())
		key, multiple := MustSucceed2(
			svc.SetByKeyOrName(ctx, "pump", "on", string(status.VariantSuccess)),
		)
		Expect(key).To(Equal("stored"))
		Expect(multiple).To(BeFalse())
		var matched []status.Status[any]
		Expect(svc.NewRetrieve[any]().
			Where(status.MatchNames[any]("pump")).
			Entries(&matched).
			Exec(ctx, db)).To(Succeed())
		Expect(matched).To(HaveExactElements(HaveField("Message", "on")))
	})
})
