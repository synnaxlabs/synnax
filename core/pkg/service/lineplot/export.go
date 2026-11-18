// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/color"
)

// prepareExport sets each line and rule color that names a custom named color to its
// fixed value, so the Core that imports the plot keeps the color.
func (s *Service) prepareExport(
	ctx context.Context,
	snap gorp.Snapshot,
	p *LinePlot,
) error {
	refs := make([]*color.Ref, 0, len(p.Lines)+len(p.Rules))
	for i := range p.Lines {
		refs = append(refs, p.Lines[i].Color)
	}
	for i := range p.Rules {
		refs = append(refs, p.Rules[i].Color)
	}
	return s.cfg.Color.InlineCustom(ctx, snap, refs)
}
