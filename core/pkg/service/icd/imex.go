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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ imex.ImportExporter = (*Service)(nil)

// Type implements imex.ImportExporter.
func (s *Service) Type() resource.Type { return resource.TypeICD }

// Match reports false: ICDs have no Console-era file shape to recognize.
func (*Service) Match(map[string]any) bool { return false }

// Decode decodes env into a new ICD that its Write persists. The key on the wire is
// discarded so every import mints a new resource; message and field keys are kept, so
// references to them survive. The Write ignores opts: an ICD has no parent and no
// group.
func (s *Service) Decode(ctx context.Context, env imex.Envelope) (imex.Write, error) {
	ic, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return nil, err
	}
	return func(
		ctx context.Context,
		tx gorp.Tx,
		_ imex.ImportOptions,
	) (resource.ID, error) {
		if err := s.NewWriter(tx).Create(ctx, &ic); err != nil {
			return resource.ID{}, err
		}
		return ResourceID(ic.Key), nil
	}, nil
}
