// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type heldLabel struct{ Key string }

func (l heldLabel) GorpKey() string { return l.Key }

// rangeHolder and noteHolder are records that hold a label.
type rangeHolder struct {
	Key   string
	Label string
}

func (h rangeHolder) GorpKey() string { return h.Key }

type noteHolder struct {
	Key   string
	Label string
}

func (h noteHolder) GorpKey() string { return h.Key }

// holdLabel returns the reference of a holder to the label at label, with onDelete.
func holdLabel[E gorp.Entry[string]](
	onDelete gorp.OnDelete,
	label func(*E) *string,
) []gorp.Reference[string, E] {
	return []gorp.Reference[string, E]{gorp.NewKeyReference(
		gorp.KeyReferenceConfig[string, E, string]{
			Name:     "label",
			Table:    string(resource.TypeLabel),
			OnDelete: onDelete,
			Kind:     gorp.Use,
			Targets:  func(h *E) []string { return []string{*label(h)} },
			Detach:   func(h *E, _ string) { *label(h) = "" },
		},
	)}
}

var _ = Describe("Holders", Ordered, func() {
	var db *gorp.DB
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, heldLabel]{
			DB:   db,
			Name: string(resource.TypeLabel),
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, rangeHolder]{
			DB:   db,
			Name: string(resource.TypeRange),
			References: holdLabel(gorp.Restrict, func(h *rangeHolder) *string {
				return &h.Label
			}),
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, noteHolder]{
			DB:   db,
			Name: "note",
			References: holdLabel(gorp.Detach, func(h *noteHolder) *string {
				return &h.Label
			}),
		}))
	})
	recordOf := func(t gorp.Target) resource.ID {
		return resource.ID{Type: resource.Type(t.Table), Key: t.Key}
	}
	open := func(types ...resource.Type) (*impact.Service, error) {
		deleters := make(map[resource.Type]impact.Deleter, len(types))
		for _, t := range types {
			deleters[t] = impact.Deleter{}
		}
		return impact.OpenService(impact.ServiceConfig{
			Deleters: deleters,
			RecordOf: recordOf,
			DB:       db,
		})
	}

	It("Should refuse to open with no Deleter for a record that restricts a delete",
		func() {
			Expect(open(resource.TypeLabel)).Error().To(MatchError(
				"[impact] - range holds label through range.label with restrict, " +
					"and no Deleter deletes it",
			))
		},
	)

	It("Should open when each record that restricts a delete has a Deleter", func() {
		MustSucceed(open(resource.TypeLabel, resource.TypeRange))
	})
})
