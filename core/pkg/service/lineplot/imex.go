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
	v5 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v5"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// lastStateVersion is the final Console state version ("5.0.0" files). A v5
// state parks the plot body under pendingUpload; earlier states embed it inline
// and ride the storage lift's legacy chain.
const lastStateVersion imex.Version = 5

var _ imex.ImportExporter = (*Service)(nil)

// Export retrieves the line plot identified by id and serializes it as an imex.Envelope
// stamped with Version. It returns query.ErrNotFound if no line plot exists for id.Key.
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
	env := imex.Envelope{Version: Version, Type: string(s.Type()), Name: lp.Name}
	if err = imex.Encode(&env, lp); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// Import decodes the envelope into a LinePlot and persists it on tx, returning the
// ontology.ID of the newly-created line plot. The exported key is discarded and a
// fresh one is generated so that importing always materializes a new resource. When
// opts.Project is non-zero the plot is created within that project exactly as a
// regular create would be. Envelopes older than Version are Console-era files —
// camelCase typed exports or console states — and are lifted forward; an envelope
// newer than Version is rejected with a path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	opts imex.ImportOptions,
) (ontology.ID, error) {
	lp, err := s.decodeImport(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	lp.Key = uuid.Nil
	// env.Name is the resolved resource name: the body's name when present, or the
	// caller-supplied file name fallback applied by the imex service.
	lp.Name = env.Name
	if err = s.NewWriter(tx).Create(ctx, opts.Project, &lp); err != nil {
		return ontology.ID{}, err
	}
	return OntologyID(lp.Key), nil
}

// stateV5 is the slice of the v5 Console state the importer needs: the plot body
// parked under pendingUpload when a line plot was never uploaded.
type stateV5 struct {
	PendingUpload *stateV5Document `json:"pending_upload"`
}

// stateV5Document mirrors the typed LinePlot body fields as they appear inside a
// v5 Console state's pendingUpload.
type stateV5Document struct {
	Title    Title    `json:"title"`
	Legend   Legend   `json:"legend"`
	Channels Channels `json:"channels"`
	Ranges   Ranges   `json:"ranges"`
	Axes     Axes     `json:"axes"`
	Lines    []Line   `json:"lines"`
	Rules    []Rule   `json:"rules"`
}

func (s *Service) decodeImport(
	ctx context.Context,
	env imex.Envelope,
) (LinePlot, error) {
	switch {
	case env.Version > Version:
		return LinePlot{}, imex.NewErrUnsupportedVersion(
			string(s.Type()), env.Version, Version,
		)
	case env.Version == Version:
		return imex.Decode[LinePlot](ctx, env)
	}
	named, err := imex.BodyNamed(ctx, env)
	if err != nil {
		return LinePlot{}, err
	}
	// Console-era typed exports (versionless) carry the current shape with
	// camelCase keys; console states never carry a name.
	if named {
		return imex.DecodeCamel[LinePlot](ctx, env)
	}
	if env.Version == lastStateVersion {
		st, err := imex.DecodeCamel[stateV5](ctx, env)
		if err != nil {
			return LinePlot{}, err
		}
		if st.PendingUpload == nil {
			return LinePlot{}, errors.Wrap(
				validate.ErrValidation, "line plot file has no body data",
			)
		}
		p := st.PendingUpload
		return LinePlot{
			Title: p.Title, Legend: p.Legend, Channels: p.Channels,
			Ranges: p.Ranges, Axes: p.Axes, Lines: p.Lines, Rules: p.Rules,
		}, nil
	}
	// v0-v4 console states embed the body inline: ride the storage lift, which
	// dispatches on the version string inside the body.
	body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
	if err != nil {
		return LinePlot{}, err
	}
	return v6.MigrateLinePlot(ctx, v5.LinePlot{Name: env.Name, Data: body})
}
