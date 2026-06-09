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
	"github.com/synnaxlabs/freighter/gorp"
	"github.com/synnaxlabs/x/errors"
	xgorp "github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type entry struct {
	ID   string
	Data string
}

func (e entry) GorpKey() string { return e.ID }

func (entry) SetOptions() []any { return nil }

var errBoom = errors.New("boom")

var _ = Describe("CreateUnaryHandlerWithTx", func() {
	It("Should commit the transaction when the handler returns nil", func(ctx SpecContext) {
		handler := gorp.CreateUnaryHandlerWithTx(
			db,
			func(ctx context.Context, tx xgorp.Tx, req entry) (entry, error) {
				if err := xgorp.NewCreate[string, entry]().
					Entry(&req).
					Exec(ctx, tx); err != nil {
					return entry{}, err
				}
				return req, nil
			})
		Expect(handler(ctx, entry{ID: "unary-commit", Data: "v"})).
			To(Equal(entry{ID: "unary-commit", Data: "v"}))
		var got entry
		Expect(
			xgorp.NewRetrieve[string, entry]().
				Where(xgorp.MatchKeys[string, entry]("unary-commit")).
				Entry(&got).Exec(ctx, db),
		).To(Succeed())
		Expect(got).To(Equal(entry{ID: "unary-commit", Data: "v"}))
	})

	It("Should roll back the transaction when the handler returns an error", func(ctx SpecContext) {
		handler := gorp.CreateUnaryHandlerWithTx(
			db,
			func(ctx context.Context, tx xgorp.Tx, req entry) (entry, error) {
				if err := xgorp.NewCreate[string, entry]().Entry(&req).
					Exec(ctx, tx); err != nil {
					return entry{}, err
				}
				return entry{}, errBoom
			})
		Expect(handler(ctx, entry{ID: "unary-rollback", Data: "v"})).
			Error().To(MatchError(errBoom))
		Expect(xgorp.NewRetrieve[string, entry]().
			Where(xgorp.MatchKeys[string, entry]("unary-rollback")).
			Entry(&entry{}).Exec(ctx, db),
		).To(MatchError(query.ErrNotFound))
	})

	It("Should return a zero RS on error", func(ctx SpecContext) {
		handler := gorp.CreateUnaryHandlerWithTx(
			db,
			func(context.Context, xgorp.Tx, entry) (entry, error) {
				return entry{ID: "leaked", Data: "leaked"}, errBoom
			})
		Expect(handler(ctx, entry{})).Error().To(MatchError(errBoom))
	})

	It("Should propagate the request to the handler unchanged", func(ctx SpecContext) {
		handler := gorp.CreateUnaryHandlerWithTx(
			db,
			func(_ context.Context, _ xgorp.Tx, req entry) (entry, error) {
				return req, nil
			})
		Expect(handler(ctx, entry{ID: "unary-propagate", Data: "data"})).
			To(Equal(entry{ID: "unary-propagate", Data: "data"}))
	})
})
