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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

var _ imex.ImportExporter = (*Service)(nil)

// Match reports whether body is a legacy Console arc state: v0-v2 files persist the
// graph inline alongside text and mode. The markers are frozen — they describe
// historical file shapes.
func (*Service) Match(body map[string]any) bool {
	_, hasGraph := body["graph"]
	_, hasMode := body["mode"]
	_, hasText := body["text"]
	return hasGraph && (hasMode || hasText)
}

// Export retrieves the arc identified by id and serializes it as an imex.Envelope
// stamped with versions.Latest. It returns query.ErrNotFound if no arc exists for
// id.Key.
func (s *Service) Export(ctx context.Context, id ontology.ID) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var a Arc
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&a).
		Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{
		Version: versions.Latest, Type: string(s.Type()), Name: a.Name,
	}
	if err = imex.Encode(&env, a); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// Import decodes the envelope into an Arc and persists it on tx, returning the
// ontology.ID of the newly-created arc. The exported key is discarded and a fresh
// one is generated so that importing always materializes a new resource. Arcs are
// not parented on import, so opts.Parent does not apply. Envelopes older than
// versions.Latest are Console-era files — camelCase typed exports or Console states —
// and are lifted forward; an envelope newer than versions.Latest is rejected with a
// path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	_ imex.ImportOptions,
) (ontology.ID, error) {
	a, err := versions.DecodeImport(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	a.Key = uuid.Nil
	// env.Name is the resolved resource name: the body's name when present, or the
	// caller-supplied file name fallback applied by the imex service.
	a.Name = env.Name
	if err = s.NewWriter(tx).Create(ctx, &a); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(a.Key), nil
}
