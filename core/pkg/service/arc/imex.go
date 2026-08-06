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

// Match reports whether body is a legacy Console Arc state, which persists the graph
// inline alongside text and mode. The markers are frozen historical file shapes.
func (*Service) Match(body map[string]any) bool {
	_, hasGraph := body["graph"]
	_, hasMode := body["mode"]
	_, hasText := body["text"]
	return hasGraph && (hasMode || hasText)
}

// Export serializes the Arc identified by id, stamping versions.Latest. It returns
// query.ErrNotFound if no Arc has id.Key.
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

// Import decodes env into an Arc and persists it on tx. The key on the wire is
// discarded so every import mints a new resource. An unknown envelope version is a
// path-scoped validation error.
//
// opts.Parent is ignored: an Arc is never parented — the writer defines the ontology
// resource with no container relationship — so there is nothing to attach it under. The
// API layer still enforces update access on the parent the caller declared.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	_ imex.ImportOptions,
) (ontology.ID, error) {
	a, err := versions.DecodeImExEnvelope(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	if err = s.NewWriter(tx).Create(ctx, &a); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(a.Key), nil
}
