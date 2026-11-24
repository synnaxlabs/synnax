// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// dataTypeRepair returns the migration that derives the data type of each stored
// calculated channel that has none from its expression. It logs each channel that is
// left without one, which the Core refuses on its next update. It reads through the
// service, so it runs after the channel table opens.
func (s *Service) dataTypeRepair() gorp.Migration {
	return migrate.New(
		"v63_channel_data_type",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var untyped []Channel
			if err := s.table.NewRetrieve().
				Where(gorp.Match(func(_ gorp.Context, c *Channel) (bool, error) {
					return c.DataType == telem.UnknownT, nil
				})).
				Entries(&untyped).
				Exec(ctx, tx); err != nil {
				return err
			}
			analyzer := NewCalculationAnalyzer(
				s.NewArcSymbolResolver(tx),
				parser.Config{AllowDashedNames: !s.ShouldValidateNames()},
			)
			repaired := make([]Channel, 0, len(untyped))
			for _, ch := range untyped {
				reason := "it is not calculated"
				if ch.IsCalculated() {
					res, err := analyzer.Analyze(ctx, ch)
					if ctxErr := ctx.Err(); ctxErr != nil {
						return ctxErr
					}
					if err == nil {
						ch.DataType = res.ChanDataType
						repaired = append(repaired, ch)
						continue
					}
					reason = err.Error()
				}
				ins.L.Warn(
					"channel has no data type, so the Core refuses its next update",
					zap.Stringer("key", ch.Key()),
					zap.String("name", ch.Name),
					zap.String("reason", reason),
				)
			}
			if len(repaired) == 0 {
				return nil
			}
			return s.table.NewCreate().Entries(&repaired).Exec(ctx, tx)
		},
	)
}
