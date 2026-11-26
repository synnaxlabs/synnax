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

type refSlot struct {
	Labels []string
	Owners []gorp.Target
}

type refPanel struct {
	Key   string
	Slots []refSlot
}

func (p refPanel) GorpKey() string { return p.Key }

// locateIn returns the path to the first value of a field of a slot of p that equals
// target. of returns the values of the field.
func locateIn[T comparable](
	p *refPanel,
	target T,
	field string,
	of func(refSlot) []T,
) []string {
	for i, s := range p.Slots {
		for j, v := range of(s) {
			if v == target {
				return []string{"slots", strconv.Itoa(i), field, strconv.Itoa(j)}
			}
		}
	}
	return nil
}

func labelsOf(s refSlot) []string { return s.Labels }

func ownersOf(s refSlot) []gorp.Target { return s.Owners }

var _ = Describe("Reference Locate", func() {
	var (
		db     *gorp.DB
		panels *gorp.Table[string, refPanel]
		// locateLabels is the Locate of the label reference, which a spec can replace.
		locateLabels func(p *refPanel, target string) []string
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		labels := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, refLabel]{
			DB:   db,
			Name: "label",
		}))
		racks := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, refRack]{
			DB:   db,
			Name: "rack",
		}))
		Expect(labels.NewCreate().
			Entries(&[]refLabel{{Key: "l1"}, {Key: "l2"}, {Key: "locked"}}).
			Exec(ctx, db)).To(Succeed())
		Expect(racks.NewCreate().Entry(&refRack{Key: 7}).Exec(ctx, db)).To(Succeed())
		locateLabels = func(p *refPanel, target string) []string {
			return locateIn(p, target, "labels", labelsOf)
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
					Unique:   true,
					Targets: func(p *refPanel) []string {
						var targets []string
						for _, s := range p.Slots {
							targets = append(targets, s.Labels...)
						}
						return targets
					},
					Locate: func(p *refPanel, target string) []string {
						return locateLabels(p, target)
					},
					Check: func(_ context.Context, _ gorp.Tx, _ *refPanel, l string) error {
						if l == "locked" {
							return errors.Wrapf(
								validate.ErrValidation,
								"label %s is locked",
								l,
							)
						}
						return nil
					},
				}),
				gorp.NewTargetReference(gorp.TargetReferenceConfig[string, refPanel]{
					Name:     "slots.owners",
					Tables:   []string{"rack"},
					OnDelete: gorp.Restrict,
					Kind:     gorp.Use,
					Targets: func(p *refPanel) []gorp.Target {
						var targets []gorp.Target
						for _, s := range p.Slots {
							targets = append(targets, s.Owners...)
						}
						return targets
					},
					Locate: func(p *refPanel, target gorp.Target) []string {
						return locateIn(p, target, "owners", ownersOf)
					},
				}),
			},
		}))
	})

	DescribeTable("Should report a failed target at its position",
		func(ctx SpecContext, held []refPanel, slots []refSlot, expected string) {
			for _, h := range held {
				Expect(panels.NewCreate().Entry(&h).Exec(ctx, db)).To(Succeed())
			}
			Expect(panels.NewCreate().
				Entry(&refPanel{Key: "p", Slots: slots}).
				Exec(ctx, db)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(expected),
			))
		},
		Entry("a key that does not exist", nil, []refSlot{
			{Labels: []string{"l1"}},
			{Labels: []string{"l2", "l9"}},
		}, "slots.1.labels.1: label l9 does not exist: validation error"),
		Entry("a key that Check refuses", nil, []refSlot{
			{Labels: []string{"l1"}},
			{Labels: []string{"locked"}},
		}, "slots.1.labels.0: label locked is locked: validation error"),
		Entry("a unique key that another entry holds", []refPanel{
			{Key: "q", Slots: []refSlot{{Labels: []string{"l2"}}}},
		}, []refSlot{
			{Labels: []string{"l1"}},
			{Labels: []string{"l2"}},
		}, "slots.1.labels.0: panel q already references label l2: validation error"),
		Entry("a target in a table the reference does not allow", nil, []refSlot{
			{
				Owners: []gorp.Target{
					{Table: "rack", Key: "7"},
					{Table: "label", Key: "l1"},
				},
			},
		}, "slots.0.owners.1: panel cannot reference label l1: validation error"),
		Entry("a target key that is not in its text form", nil, []refSlot{
			{},
			{Owners: []gorp.Target{{Table: "rack", Key: "07"}}},
		}, `slots.1.owners.0: "07" is not a key of rack: validation error`),
	)

	It("Should fail with no validation error when Locate finds no place for a target",
		func(ctx SpecContext) {
			locateLabels = func(*refPanel, string) []string { return nil }
			err := panels.NewCreate().Entry(&refPanel{
				Key:   "p",
				Slots: []refSlot{{Labels: []string{"l9"}}},
			}).Exec(ctx, db)
			Expect(errors.Is(err, validate.ErrValidation)).To(BeFalse())
			Expect(err).To(MatchError(
				"reference slots.labels found no place that holds label l9",
			))
		},
	)
})
