// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd/versions"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
)

var _ imex.ImportExporter = (*Service)(nil)

// Match reports false: ICDs have no Console-era file shape to recognize.
func (*Service) Match(map[string]any) bool { return false }

// Import decodes env into a new ICD and persists it on tx. The key on the wire is
// discarded so every import mints a new resource; message and field keys are kept, so
// references to them survive. opts.Parent is ignored: an ICD is never parented.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	_ imex.ImportOptions,
) (ontology.ID, error) {
	ic, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	if err = s.NewWriter(tx).Create(ctx, &ic); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(ic.Key), nil
}
