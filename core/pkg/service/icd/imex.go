// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/library/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ imex.ImportExporter = (*Service)(nil)

// Match reports false: libraries have no Console-era file shape to recognize.
func (*Service) Match(map[string]any) bool { return false }

// Import decodes env into a new Library and persists it on tx. The key on the wire is
// discarded so every import mints a new resource; entry and field keys are kept, so
// references between entries survive. opts.Parent is ignored: a library is never
// parented.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	_ imex.ImportOptions,
) (ontology.ID, error) {
	l, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	if err = s.NewWriter(tx).Create(ctx, &l); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(l.Key), nil
}
