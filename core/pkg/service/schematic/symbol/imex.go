// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.ImportExporter = (*Service)(nil)

// Match reports whether body is a legacy Console symbol file, which nests the spec in a
// data object with an inline svg. The marker is a frozen historical file shape.
func (*Service) Match(body map[string]any) bool {
	data, ok := body["data"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = data["svg"]
	return ok
}

// Export serializes the symbol identified by id, stamping versions.Latest. It returns
// query.ErrNotFound if no symbol has id.Key.
func (s *Service) Export(ctx context.Context, id ontology.ID) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var sym Symbol
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&sym).
		Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{
		Version: versions.Latest, Type: string(s.Type()), Name: sym.Name,
	}
	if err = imex.Encode(&env, sym); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// Import decodes env into a Symbol created under opts.Parent, which must be a group.
// The key on the wire is discarded so every import mints a new resource. An unknown
// envelope version is a path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (ontology.ID, error) {
	if _, err := group.KeyFromOntologyID(opts.Parent); err != nil {
		return ontology.ID{}, validate.PathedError(err, "parent")
	}
	sym, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	if err = s.NewWriter(tx).Create(ctx, &sym, opts.Parent); err != nil {
		return ontology.ID{}, err
	}
	return sym.OntologyID(), nil
}
