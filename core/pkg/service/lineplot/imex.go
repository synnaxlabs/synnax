// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

var (
	_ imex.ImportExporter = (*Service)(nil)
	_ imex.Matcher        = (*Service)(nil)
)

// Match reports whether body is a legacy Console line plot state: v0-v4 files persist
// the plot body inline (axes/channels), v5 carries selectedRules/hiddenLines alongside
// an optional pendingUpload. The markers are frozen — they describe historical file
// shapes.
func (*Service) Match(body map[string]any) bool {
	_, hasAxes := body["axes"]
	_, hasChannels := body["channels"]
	_, hasSelectedRules := body["selectedRules"]
	_, hasHiddenLines := body["hiddenLines"]
	return (hasAxes && hasChannels) || hasSelectedRules || hasHiddenLines
}

// Export retrieves the line plot identified by id and serializes it as an
// imex.Envelope stamped with versions.Latest. It returns query.ErrNotFound if no
// line plot exists for id.Key.
func (s *Service) Export(ctx context.Context, id ontology.ID) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var lp LinePlot
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&lp).
		Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{
		Version: versions.Latest, Type: string(s.Type()), Name: lp.Name,
	}
	if err = imex.Encode(&env, lp); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// Import decodes the envelope into a LinePlot and persists it on tx, returning the
// ontology.ID of the newly-created line plot. The exported key is discarded and a
// fresh one is generated so that importing always materializes a new resource. Line
// plots are project children, so opts.Parent is required and must be a project; the
// plot
// is then created within it exactly as a regular create would be. Envelopes older
// than versions.Latest are Console-era files — camelCase typed exports or Console
// states — and are lifted forward; an envelope newer than versions.Latest is
// rejected with a path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (ontology.ID, error) {
	if opts.Parent.IsZero() {
		return ontology.ID{}, validate.PathedError(validate.ErrRequired, "parent")
	}
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
	lp, err := versions.DecodeImport(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	lp.Key = uuid.Nil
	// env.Name is the resolved resource name: the body's name when present, or the
	// caller-supplied file name fallback applied by the imex service.
	lp.Name = env.Name
	if err = s.NewWriter(tx).Create(ctx, proj, &lp); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(lp.Key), nil
}
