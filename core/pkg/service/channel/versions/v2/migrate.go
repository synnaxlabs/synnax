// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"encoding/json"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	"go.uber.org/zap"
)

// CalculationKVPrefix is the KV key prefix under which StageCalculations stages the
// calculation of each stored channel. The rest of the key is the channel key. The
// calculation service builds its records from the staged entries and deletes them.
const CalculationKVPrefix = "sy_channel_calculation/"

// StagedCalculation is what StageCalculations stages for a channel.
type StagedCalculation struct {
	// Expression is the expression of the channel.
	Expression string `json:"expression"`
	// Operations are the operations of the channel.
	Operations []Operation `json:"operations"`
}

// StageCalculations copies the expression and operations of each stored channel that
// has an expression into an entry under CalculationKVPrefix, before Migration drops
// both fields.
var StageCalculations = migrate.New(
	"v63_stage_calculations",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		var channels []v1.Channel
		if err := gorp.NewRetrieve[v1.Key, v1.Channel]().
			Where(gorp.Match(func(_ gorp.Context, c *v1.Channel) (bool, error) {
				return c.IsCalculated(), nil
			})).
			Entries(&channels).
			Exec(ctx, tx); err != nil {
			return err
		}
		for _, c := range channels {
			blob, err := json.Marshal(StagedCalculation{
				Expression: c.Expression,
				Operations: c.Operations,
			})
			if err != nil {
				return err
			}
			key := []byte(CalculationKVPrefix + c.Key().String())
			if err = tx.Set(ctx, key, blob); err != nil {
				return err
			}
		}
		return nil
	},
)

// Migration drops the internal flag of each stored channel and logs each channel that
// held it. It drops the expression and operations of each channel, which
// StageCalculations staged.
var Migration = migrate.NewInstrumentedEntry(
	"v63_drop_internal",
	func(ctx context.Context, old v1.Channel, ins alamos.Instrumentation) (Channel, error) {
		if old.Internal {
			ins.L.Warn(
				"dropped the internal flag of a channel",
				zap.Stringer("key", old.Key()),
				zap.String("name", old.Name),
			)
		}
		return autoMigrateChannel(ctx, old)
	},
)
