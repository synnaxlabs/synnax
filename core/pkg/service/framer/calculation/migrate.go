// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/gorp"
	"go.uber.org/zap"
)

func (s *Service) migrateChannels(ctx context.Context) error {
	var legacyCalculations []channel.Channel
	if err := s.cfg.Channel.NewRetrieve().
		WhereLegacyCalculated().
		Entries(&legacyCalculations).
		Exec(ctx, nil); err != nil {
		return err
	}
	return s.cfg.DB.WithTx(ctx, func(tx gorp.Tx) error {
		writer := s.cfg.Channel.NewWriter(tx)
		for _, calc := range legacyCalculations {
			if _, err := compile(ctx, CalculatorConfig{
				ChannelSvc: s.cfg.Channel,
				Channel:    calc,
				Resolver:   s.cfg.Arc.SymbolResolver(),
			}); err == nil {
				calc.Requires = nil
				if err = writer.Create(ctx, &calc); err != nil {
					return err
				}

				s.cfg.L.Info(
					"successfully migrated legacy calculation",
					zap.String("calc", calc.Name),
					zap.Stringer("key", calc.Key()),
				)
			} else {
				s.cfg.L.Warn(
					"failed to compile legacy calculation",
					zap.Error(err),
					zap.String("expression", calc.Expression),
				)
			}
		}
		return nil
	})
}
