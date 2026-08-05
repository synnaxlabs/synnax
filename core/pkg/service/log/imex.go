// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.ImportExporter = (*Service)(nil)

// Match reports whether body is a legacy Console log state, which persists channels as
// an array (bare keys at v0, config objects at v1); no other resource's state does.
// The marker is frozen — it describes historical file shapes.
func (*Service) Match(body map[string]any) bool {
	_, ok := body["channels"].([]any)
	return ok
}

// Export retrieves the log identified by id and serializes it as an imex.Envelope
// stamped with versions.Latest. It returns query.ErrNotFound if no log exists for
// id.Key.
func (s *Service) Export(ctx context.Context, id ontology.ID) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var l Log
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&l).
		Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{Version: versions.Latest, Type: string(s.Type()), Name: l.Name}
	if err = imex.Encode(&env, l); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// Import decodes the envelope into a Log and persists it on tx, returning the
// ontology.ID of the newly-created log. The exported key is discarded and a fresh one
// is generated so that importing always materializes a new resource rather than
// overwriting an existing log with a colliding key. Logs are project children, so
// opts.Parent
// must be a project; the log is then created within it exactly
// as a regular create would be. Envelopes older than versions.Latest are legacy
// camelCase Console exports and are lifted forward through the migration chain; an
// envelope newer than versions.Latest is rejected with a path-scoped validation
// error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (ontology.ID, error) {
	if opts.Parent.Type != ontology.ResourceTypeProject {
		return ontology.ID{}, validate.PathedError(
			errors.Wrapf(
				validate.ErrValidation,
				"parent must be a project, got %q",
				opts.Parent.Type,
			),
			"parent",
		)
	}
	proj, err := uuid.Parse(opts.Parent.Key)
	if err != nil {
		return ontology.ID{}, validate.PathedError(
			errors.Wrapf(
				validate.ErrValidation, "invalid project key %q", opts.Parent.Key,
			),
			"parent",
		)
	}
	l, err := versions.DecodeImport(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	l.Key = uuid.Nil
	// env.Name is the resolved resource name: the body's name when present, or the
	// caller-supplied file name fallback applied by the imex service. The two agree
	// whenever the body carries a name, so this only matters for nameless bodies.
	l.Name = env.Name
	if err = s.NewWriter(tx).Create(ctx, proj, &l); err != nil {
		return ontology.ID{}, err
	}
	return l.OntologyID(), nil
}
