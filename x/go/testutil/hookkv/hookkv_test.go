// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package hookkv_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/hookkv"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var errRefused = errors.New("refused")

func refuse(func() error) error { return errRefused }

func set(ctx context.Context, db kv.DB, key string) error {
	tx := db.OpenTx()
	defer func() { Expect(tx.Close()).To(Succeed()) }()
	Expect(tx.Set(ctx, []byte(key), []byte("value"))).To(Succeed())
	return tx.Commit(ctx)
}

func stored(ctx context.Context, db kv.DB, key string) bool {
	_, closer, err := db.Get(ctx, []byte(key))
	if errors.Is(err, query.ErrNotFound) {
		return false
	}
	Expect(err).ToNot(HaveOccurred())
	Expect(closer.Close()).To(Succeed())
	return true
}

var _ = Describe("DB", func() {
	var db *hookkv.DB
	BeforeEach(func() { db = hookkv.Wrap(DeferClose(memkv.New())) })

	It("Should commit a transaction that no hook intercepts", func(ctx SpecContext) {
		Expect(set(ctx, db, "a")).To(Succeed())
		Expect(stored(ctx, db, "a")).To(BeTrue())
	})

	It("Should run a commit through the hook of its context", func(ctx SpecContext) {
		Expect(set(hookkv.WithHook(ctx, refuse), db, "a")).To(MatchError(errRefused))
		Expect(stored(ctx, db, "a")).To(BeFalse())
		Expect(set(ctx, db, "b")).To(Succeed())
	})

	It("Should run a commit that sets a marked key through the key hook",
		func(ctx SpecContext) {
			remove := db.HookKey([]byte("mark"), refuse)
			Expect(set(ctx, db, "x-mark-x")).To(MatchError(errRefused))
			Expect(stored(ctx, db, "x-mark-x")).To(BeFalse())
			Expect(set(ctx, db, "other")).To(Succeed())
			remove()
			Expect(set(ctx, db, "x-mark-x")).To(Succeed())
		},
	)

	It("Should run the hook of the context before a key hook", func(ctx SpecContext) {
		var keyed bool
		db.HookKey([]byte("mark"), func(commit func() error) error {
			keyed = true
			return commit()
		})
		Expect(set(hookkv.WithHook(ctx, refuse), db, "mark")).To(MatchError(errRefused))
		Expect(keyed).To(BeFalse())
	})

	Describe("Hold", func() {
		It("Should hold the first commit until release, and pass later commits",
			func(ctx SpecContext) {
				held, release := make(chan struct{}), make(chan struct{})
				db.HookKey([]byte("mark"), hookkv.Hold(held, release))
				first := make(chan error, 1)
				go func() {
					defer GinkgoRecover()
					first <- set(ctx, db, "mark-1")
				}()
				Eventually(held).Should(BeClosed())
				Consistently(first).ShouldNot(Receive())
				Expect(stored(ctx, db, "mark-1")).To(BeFalse())
				close(release)
				Eventually(first).Should(Receive(BeNil()))
				Expect(set(ctx, db, "mark-2")).To(Succeed())
				Expect(stored(ctx, db, "mark-1")).To(BeTrue())
			},
		)
	})
})
