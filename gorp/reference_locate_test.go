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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

type refSlot struct{ Labels []string }

type refPanel struct {
	Key   string
	Slots []refSlot
}

func (p refPanel) GorpKey() string { return p.Key }

var _ = Describe("Reference Locate", func() {
	var (
		db     *gorp.DB
		panels *gorp.Table[string, refPanel]
		locate func(p *refPanel, target string) []string
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		labels := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
			DB:   db,
			Name: "label",
		}))
		Expect(labels.NewCreate().Entry(&refLabel{Key: "l1"}).Exec(ctx, db)).
			To(Succeed())
		locate = func(p *refPanel, target string) []string {
			for i, s := range p.Slots {
				for j, l := range s.Labels {
					if l == target {
						return []string{
							"slots",
							strconv.Itoa(i),
							"labels",
							strconv.Itoa(j),
						}
					}
				}
			}
			return nil
		}
		panels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refPanel]{
			DB:   db,
			Name: "panel",
			References: []gorp.Reference[string, refPanel]{
				gorp.NewKeyReference(gorp.KeyReferenceConfig[string, refPanel, string]{
					Name:     "slots.labels",
					Table:    "label",
					OnDelete: gorp.Restrict,
					Kind:     gorp.Use,
					Targets: func(p *refPanel) []string {
						var targets []string
						for _, s := range p.Slots {
							targets = append(targets, s.Labels...)
						}
						return targets
					},
					Locate: func(p *refPanel, target string) []string {
						return locate(p, target)
					},
				}),
			},
		}))
	})
	panel := refPanel{
		Key:   "p",
		Slots: []refSlot{{Labels: []string{"l1"}}, {Labels: []string{"l1", "l9"}}},
	}

	It(
		"Should name the position of a target that does not exist",
		func(ctx SpecContext) {
			err := panels.NewCreate().Entry(&panel).Exec(ctx, db)
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err.Error()).To(Equal(
				"slots.1.labels.1: label l9 does not exist: validation error",
			))
		},
	)

	It("Should fail with no validation error when Locate finds no place for a target",
		func(ctx SpecContext) {
			locate = func(*refPanel, string) []string { return nil }
			err := panels.NewCreate().Entry(&panel).Exec(ctx, db)
			Expect(errors.Is(err, validate.ErrValidation)).To(BeFalse())
			Expect(err).To(MatchError(
				"reference slots.labels found no place that holds label l9",
			))
		},
	)
})
