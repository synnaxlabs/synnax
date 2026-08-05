// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/table/versions"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.ImportExporter = (*Service)(nil)

// Match reports whether body is a legacy Console table state, which persists the
// structural model inline under layout and cells. The markers are frozen historical
// file shapes.
func (*Service) Match(body map[string]any) bool {
	_, hasLayout := body["layout"]
	_, hasCells := body["cells"]
	return hasLayout && hasCells
}

// Export serializes the table identified by id, stamping versions.Latest. It returns
// query.ErrNotFound if no table has id.Key.
func (s *Service) Export(ctx context.Context, id ontology.ID) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var t Table
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&t).
		Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{Version: versions.Latest, Type: string(s.Type()), Name: t.Name}
	if err = imex.Encode(&env, t); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// Import decodes env into a Table created under opts.Parent, which must be a project.
// The key on the wire is discarded so every import mints a new resource. An unknown
// envelope version is a path-scoped validation error.
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
	t, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	if err = s.NewWriter(tx).Create(ctx, proj, &t); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(t.Key), nil
}
