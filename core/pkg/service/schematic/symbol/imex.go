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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.ImportExporter = (*Service)(nil)

// Match reports whether body is a legacy Console symbol file: main-era exports carry
// the symbol spec as a data object with an inline svg, and no other typeless file
// family nests its payload that way. The marker is frozen — it describes historical
// file shapes; current exports carry a type header and never reach matching.
func (*Service) Match(body map[string]any) bool {
	data, ok := body["data"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = data["svg"]
	return ok
}

// Export retrieves the symbol identified by id and serializes it as an imex.Envelope
// stamped with versions.Latest. It returns query.ErrNotFound if no symbol exists for
// id.Key.
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

// Import decodes the envelope into a Symbol and persists it on tx, returning the
// ontology.ID of the newly-created symbol. The exported key is discarded and a fresh
// one is generated so that importing always materializes a new resource. Symbols are
// parented into groups: opts.Parent must be a group. Envelopes below versions.Latest
// are Console-written camelCase files; an envelope newer than versions.Latest is
// rejected with a path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (ontology.ID, error) {
	if opts.Parent.Type != ontology.ResourceTypeGroup {
		return ontology.ID{}, validate.PathedError(
			errors.Wrapf(
				validate.ErrValidation,
				"symbol parent must be a group, got %q", opts.Parent.Type,
			),
			"parent",
		)
	}
	sym, err := versions.DecodeImport(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	sym.Key = uuid.Nil
	// env.Name is the resolved resource name: the body's name when present, or the
	// caller-supplied file name fallback applied by the imex service.
	sym.Name = env.Name
	if err = s.NewWriter(tx).Create(ctx, &sym, opts.Parent); err != nil {
		return ontology.ID{}, err
	}
	return sym.OntologyID(), nil
}
