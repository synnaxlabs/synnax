// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	"github.com/synnaxlabs/synnax/pkg/service/channel/resolver"
)

// NewAnalyzer returns an analyzer that resolves channels through snap.
func (s *Service) NewAnalyzer(snap gorp.Snapshot) *compiler.Analyzer {
	return compiler.NewAnalyzer(
		resolver.New(s.channels, snap),
		s.stl,
		parser.Config{AllowDashedNames: resolver.AllowDashedNames(s.channels)},
	)
}

// Compile compiles a calculation as compiler.Compile does, with the channel service and
// the standard library symbols of s, so cfgs need not set them.
func (s *Service) Compile(
	ctx context.Context,
	db *gorp.DB,
	cfgs ...compiler.Config,
) (compiler.Module, error) {
	return compiler.Compile(ctx, db, append(
		[]compiler.Config{{ChannelService: s.channels, STL: s.stl}},
		cfgs...,
	)...)
}

// PreProcess analyzes a calculation as compiler.PreProcess does, with the channel
// service and the standard library symbols of s in place of those of cfg.
func (s *Service) PreProcess(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg compiler.Config,
) (arc.Program, error) {
	cfg.ChannelService = s.channels
	cfg.STL = s.stl
	return compiler.PreProcess(ctx, snap, cfg)
}
