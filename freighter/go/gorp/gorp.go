// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package gorp provides freighter handler adapters that execute the wrapped handler
// inside a single Gorp transaction.
package gorp

import (
	"context"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/gorp"
)

// CreateUnaryHandler wraps a unary handler that needs to execute inside a single
// [gorp.Tx] to a [freighter.UnaryHandler]. The transaction is committed when handle
// returns nil and rolled back otherwise. On any error a zero RS is returned.
func CreateUnaryHandler[RQ, RS freighter.Payload](
	db *gorp.DB,
	handle func(context.Context, gorp.Tx, RQ) (RS, error),
) freighter.UnaryHandler[RQ, RS] {
	return func(ctx context.Context, req RQ) (RS, error) {
		var res RS
		if err := db.WithTx(ctx, func(tx gorp.Tx) error {
			var err error
			res, err = handle(ctx, tx, req)
			return err
		}); err != nil {
			var zero RS
			return zero, err
		}
		return res, nil
	}
}
