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
	v6 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v6"
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/validate"
)

// lastStateVersion is the final Console state version ("6.0.0" files). A v6
// state parks the document under pendingUpload; earlier states embed it inline
// and ride the storage lift's legacy chain.
const lastStateVersion imex.Version = 6

var (
	_ imex.ImportExporter = (*Service)(nil)
	_ imex.Matcher        = (*Service)(nil)
)

// Match reports whether body is a legacy Console schematic state: v0-v5 files persist
// the document inline (nodes/edges/props), v6 carries controlStatus alongside an
// optional pendingUpload. The markers are frozen — they describe historical file
// shapes.
func (s *Service) Match(body map[string]any) bool {
	_, hasNodes := body["nodes"]
	_, hasProps := body["props"]
	_, hasControlStatus := body["controlStatus"]
	return (hasNodes && hasProps) || hasControlStatus
}

// Export retrieves the schematic identified by id and serializes it as an imex.Envelope
// stamped with Version. It returns query.ErrNotFound if no schematic exists for id.Key.
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
	env := imex.Envelope{Version: Version, Type: string(s.Type()), Name: sch.Name}
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
// older than Version are Console-era files — camelCase typed exports or console
// states — and are lifted forward; an envelope newer than Version is rejected with a
// path-scoped validation error.
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
	sch, err := s.decodeImport(ctx, env)
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

// stateV6 is the slice of the v6 Console state the importer needs: the document
// body parked under pendingUpload when a schematic was never uploaded. The tag
// is camelCase because the file was written by the Console.
type stateV6 struct {
	PendingUpload *consoleDocument `json:"pendingUpload" msgpack:"pendingUpload"`
}

// consoleNode mirrors Node as Console-written files serialize it: camelCase
// keys. Frozen; Console files no longer evolve.
type consoleNode struct {
	Key      string     `json:"key" msgpack:"key"`
	Position spatial.XY `json:"position" msgpack:"position"`
	ZIndex   int16      `json:"zIndex" msgpack:"zIndex"`
}

// consoleDocument mirrors the typed Schematic body fields as Console-written
// files serialize them: the typed export's top level, and the v6 state's
// pendingUpload. Configs values are opaque user JSON and pass through as-is.
type consoleDocument struct {
	Snapshot bool                           `json:"snapshot" msgpack:"snapshot"`
	Nodes    []consoleNode                  `json:"nodes" msgpack:"nodes"`
	Edges    []Edge                         `json:"edges" msgpack:"edges"`
	Configs  map[string]msgpack.EncodedJSON `json:"configs" msgpack:"configs"`
}

// schematic lifts the Console document into the current Schematic shape.
func (d consoleDocument) schematic() Schematic {
	nodes := make([]Node, len(d.Nodes))
	for i, n := range d.Nodes {
		nodes[i] = Node{Key: n.Key, Position: n.Position, ZIndex: n.ZIndex}
	}
	return Schematic{
		Snapshot: d.Snapshot, Nodes: nodes, Edges: d.Edges, Configs: d.Configs,
	}
}

func (s *Service) decodeImport(
	ctx context.Context,
	env imex.Envelope,
) (Schematic, error) {
	switch {
	case env.Version > Version:
		return Schematic{}, imex.NewErrUnsupportedVersion(
			string(s.Type()), env.Version, Version,
		)
	case env.Version == Version:
		return imex.Decode[Schematic](ctx, env)
	}
	named, err := imex.BodyNamed(ctx, env)
	if err != nil {
		return Schematic{}, err
	}
	// Console-era typed exports ("6.0.0"-stamped or versionless) carry the current
	// shape with camelCase keys; console states never carry a name.
	if named {
		doc, err := imex.Decode[consoleDocument](ctx, env)
		if err != nil {
			return Schematic{}, err
		}
		return doc.schematic(), nil
	}
	if env.Version == lastStateVersion {
		st, err := imex.Decode[stateV6](ctx, env)
		if err != nil {
			return Schematic{}, err
		}
		if st.PendingUpload == nil {
			return Schematic{}, errors.Wrap(
				validate.ErrValidation, "schematic file has no document data",
			)
		}
		return st.PendingUpload.schematic(), nil
	}
	// v0-v5 console states embed the document inline: ride the storage lift, which
	// dispatches on the version string inside the body.
	type snapshotPeek struct {
		Snapshot bool `json:"snapshot"`
	}
	peek, err := imex.Decode[snapshotPeek](ctx, env)
	if err != nil {
		return Schematic{}, err
	}
	body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
	if err != nil {
		return Schematic{}, err
	}
	return v7.MigrateSchematic(ctx, v6.Schematic{
		Name: env.Name, Snapshot: peek.Snapshot, Data: body,
	})
}
