// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ imex.ImportExporter = (*Service)(nil)

// Type implements imex.ImportExporter.
func (s *Service) Type() resource.Type { return resource.TypeArc }

// Match reports whether body is a legacy Console Arc state, which persists the graph
// inline alongside text and mode. The markers are frozen historical file shapes.
func (*Service) Match(body map[string]any) bool {
	_, hasGraph := body["graph"]
	_, hasMode := body["mode"]
	_, hasText := body["text"]
	return hasGraph && (hasMode || hasText)
}

// Decode decodes env into an Arc that its Write persists. The key on the wire is
// discarded so every import mints a new resource. An unknown envelope version is a
// path-scoped validation error.
//
// The Write ignores opts: the writer puts every new Arc in the root directory.
func (s *Service) Decode(ctx context.Context, env imex.Envelope) (imex.Write, error) {
	a, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return nil, err
	}
	return func(
		ctx context.Context,
		tx gorp.Tx,
		_ imex.ImportOptions,
	) (resource.ID, error) {
		if err := s.NewWriter(tx).Create(ctx, &a); err != nil {
			return resource.ID{}, err
		}
		return a.ResourceID(), nil
	}, nil
}
