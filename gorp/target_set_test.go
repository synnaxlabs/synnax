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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type hookTarget struct{ Key string }

func (t hookTarget) GorpKey() string { return t.Key }

type hookHolderA struct {
	Key    string
	Target string
}

func (h hookHolderA) GorpKey() string { return h.Key }

type hookHolderB struct {
	Key    string
	Target string
}

func (h hookHolderB) GorpKey() string { return h.Key }

// openBreakerHolder opens a table of E, named name, whose entries use a target and
// whose hook records the key of each entry it runs for.
func openBreakerHolder[E gorp.Entry[string]](
	ctx SpecContext,
	db *gorp.DB,
	name string,
	targetOf func(*E) string,
	ran *[]string,
) *gorp.Table[string, E] {
	GinkgoHelper()
	ref := gorp.NewKeyReference(gorp.KeyReferenceConfig[string, E, string]{
		Name:     "target",
		Table:    "hook_target",
		OnDelete: gorp.Restrict,
		Kind:     gorp.Use,
		Targets: func(e *E) []string {
			if t := targetOf(e); t != "" {
				return []string{t}
			}
			return nil
		},
	})
	return MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, E]{
		DB:         db,
		Name:       name,
		References: []gorp.Reference[string, E]{ref},
		OnTargetSet: map[string]func(context.Context, gorp.Tx, E) error{
			"hook_target": func(_ context.Context, _ gorp.Tx, e E) error {
				*ran = append(*ran, e.GorpKey())
				return nil
			},
		},
	}))
}

var _ = Describe("OnTargetSet across two holder tables", func() {
	It("Should run the hook of a later table when an earlier one holds no entry", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		targets := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, hookTarget]{
			DB:   db,
			Name: "hook_target",
		}))
		var ran []string
		openBreakerHolder(
			ctx, db, "hook_a",
			func(h *hookHolderA) string { return h.Target },
			&ran,
		)
		holdersB := openBreakerHolder(
			ctx, db, "hook_b",
			func(h *hookHolderB) string { return h.Target },
			&ran,
		)
		Expect(targets.NewCreate().Entry(&hookTarget{Key: "t1"}).Exec(ctx, db)).
			To(Succeed())
		Expect(holdersB.NewCreate().
			Entry(&hookHolderB{Key: "b1", Target: "t1"}).
			Exec(ctx, db)).To(Succeed())
		ran = nil
		Expect(targets.NewCreate().Entry(&hookTarget{Key: "t1"}).Exec(ctx, db)).
			To(Succeed())
		Expect(ran).To(ConsistOf("b1"))
	})
})
