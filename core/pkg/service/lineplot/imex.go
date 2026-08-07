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

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.ImportExporter = (*Service)(nil)

// Match reports whether body is a legacy Console line plot state, which persists the
// plot body inline under axes and channels. The markers are frozen historical file
// shapes.
func (*Service) Match(body map[string]any) bool {
	_, hasAxes := body["axes"]
	_, hasChannels := body["channels"]
	return hasAxes && hasChannels
}

// Import decodes env into a LinePlot created under opts.Parent, which must be a
// project. The key on the wire is discarded so every import mints a new resource. An
// unknown envelope version is a path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (ontology.ID, error) {
	proj, err := project.KeyFromOntologyID(opts.Parent)
	if err != nil {
		return ontology.ID{}, validate.PathedError(err, "parent")
	}
	lp, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	if err = s.NewWriter(tx).Create(ctx, proj, &lp); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(lp.Key), nil
}
