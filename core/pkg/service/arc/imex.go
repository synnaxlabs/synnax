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
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

var _ imex.ImportExporter = (*Service)(nil)

// Export retrieves the arc identified by id and serializes it as an imex.Envelope
// stamped with Version. It returns query.ErrNotFound if no arc exists for id.Key.
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
	env := imex.Envelope{Version: Version, Type: string(s.Type()), Name: a.Name}
	if err = imex.Encode(&env, a); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// Import decodes the envelope into an Arc and persists it on tx, returning the
// ontology.ID of the newly-created arc. The exported key is discarded and a fresh
// one is generated so that importing always materializes a new resource. Arcs are
// not project children, so opts.Project does not apply. Envelopes older than
// Version are Console-era files — camelCase typed exports or console states — and
// are lifted forward; an envelope newer than Version is rejected with a
// path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	_ imex.ImportOptions,
) (ontology.ID, error) {
	a, err := s.decodeImport(ctx, env)
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

// stateV3 is the slice of the "3.0.0" Console state the importer needs: the
// document body parked under pendingUpload when an arc was never uploaded. v3
// states share the current envelope version; the two are told apart by the typed
// export's top-level name.
type stateV3 struct {
	PendingUpload *stateV3Document `json:"pending_upload"`
}

// stateV3Document mirrors the typed Arc body fields as they appear inside a v3
// Console state's pendingUpload.
type stateV3Document struct {
	Mode  Mode        `json:"mode"`
	Graph graph.Graph `json:"graph"`
	Text  text.Text   `json:"text"`
}

func (s *Service) decodeImport(ctx context.Context, env imex.Envelope) (Arc, error) {
	if env.Version > Version {
		return Arc{}, imex.NewErrUnsupportedVersion(
			string(s.Type()), env.Version, Version,
		)
	}
	named, err := imex.BodyNamed(ctx, env)
	if err != nil {
		return Arc{}, err
	}
	// Typed exports always carry a top-level name; console states never do.
	// Server exports stamp the current version with snake_case keys; Console
	// typed exports are versionless with camelCase keys.
	if named {
		if env.Version == Version {
			return imex.Decode[Arc](ctx, env)
		}
		return imex.DecodeCamel[Arc](ctx, env)
	}
	if env.Version == Version {
		st, err := imex.DecodeCamel[stateV3](ctx, env)
		if err != nil {
			return Arc{}, err
		}
		if st.PendingUpload == nil {
			return Arc{}, errors.Wrap(
				validate.ErrValidation, "arc file has no graph data",
			)
		}
		p := st.PendingUpload
		return Arc{Mode: p.Mode, Graph: p.Graph, Text: p.Text}, nil
	}
	// "0.0.0".."2.0.0" console states embed the graph inline.
	body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
	if err != nil {
		return Arc{}, err
	}
	doc, err := legacy.Migrate(body)
	if err != nil {
		return Arc{}, err
	}
	mode := Mode(doc.Mode)
	if mode == "" {
		mode = ModeGraph
	}
	return Arc{Mode: mode, Graph: doc.Graph, Text: doc.Text}, nil
}
