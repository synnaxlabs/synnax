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
)

var _ imex.ImportExporter = (*Service)(nil)

// Match reports whether body is a legacy Console table state: v0 files persist the
// structural model inline (layout/cells), v1 carries selectedCells/hideIndicators
// alongside an optional pendingUpload. The markers are frozen — they describe
// historical file shapes.
func (*Service) Match(body map[string]any) bool {
	_, hasLayout := body["layout"]
	_, hasCells := body["cells"]
	_, hasSelectedCells := body["selectedCells"]
	_, hasHideIndicators := body["hideIndicators"]
	return (hasLayout && hasCells) || hasSelectedCells || hasHideIndicators
}

// Export retrieves the table identified by id and serializes it as an imex.Envelope
// stamped with versions.Latest. It returns query.ErrNotFound if no table exists for
// id.Key.
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

// Import decodes the envelope into a Table and persists it on tx, returning the
// ontology.ID of the newly-created table. The exported key is discarded and a fresh one
// is generated so that importing always materializes a new resource. Tables are project
// children, so opts.Parent must be a project; the table is then created within it
// exactly as a regular create would be. Envelopes older than versions.Latest are
// Console-era files — camelCase typed exports or Console states — and are lifted
// forward; an envelope newer than versions.Latest is rejected with a path-scoped
// validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (ontology.ID, error) {
	proj, err := project.ParentKey(opts)
	if err != nil {
		return ontology.ID{}, err
	}
	t, err := versions.DecodeImport(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	t.Key = uuid.Nil
	// env.Name is the resolved resource name: the body's name when present, or the
	// caller-supplied file name fallback applied by the imex service.
	t.Name = env.Name
	if err = s.NewWriter(tx).Create(ctx, proj, &t); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(t.Key), nil
}
