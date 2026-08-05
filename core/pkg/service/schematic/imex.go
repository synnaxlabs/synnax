// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
	"github.com/synnaxlabs/x/gorp"
)

var (
	_ imex.ImportExporter = (*Service)(nil)
	_ imex.Matcher        = (*Service)(nil)
)

// Match reports whether body is a legacy Console schematic state: v0-v5 files persist
// the document inline (nodes/edges/props), v6 carries controlStatus alongside an
// optional pendingUpload. The markers are frozen — they describe historical file
// shapes.
func (*Service) Match(body map[string]any) bool {
	_, hasNodes := body["nodes"]
	_, hasProps := body["props"]
	_, hasControlStatus := body["controlStatus"]
	return (hasNodes && hasProps) || hasControlStatus
}

// Export retrieves the schematic identified by id and serializes it as an
// imex.Envelope stamped with versions.Latest. It returns query.ErrNotFound if no
// schematic exists for id.Key.
func (s *Service) Export(ctx context.Context, id ontology.ID) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var sch Schematic
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&sch).
		Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{
		Version: versions.Latest, Type: string(s.Type()), Name: sch.Name,
	}
	if err = imex.Encode(&env, sch); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// Import decodes the envelope into a Schematic and persists it on tx, returning the
// ontology.ID of the newly-created schematic. The exported key is discarded and a
// fresh one is generated so that importing always materializes a new resource.
// Schematics are project children, so a non-zero opts.Parent must be a project; the
// schematic is then created within it exactly as a regular create would be. Envelopes
// older than versions.Latest are Console-era files — camelCase typed exports or
// Console states — and are lifted forward; an envelope newer than versions.Latest is
// rejected with a path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (ontology.ID, error) {
	proj, err := opts.ProjectKey()
	if err != nil {
		return ontology.ID{}, err
	}
	sch, err := versions.DecodeImport(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	sch.Key = uuid.Nil
	// env.Name is the resolved resource name: the body's name when present, or the
	// caller-supplied file name fallback applied by the imex service.
	sch.Name = env.Name
	if err = s.NewWriter(tx).Create(ctx, proj, &sch); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(sch.Key), nil
}
