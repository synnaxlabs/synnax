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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/spatial"
)

var _ imex.ImportExporter = (*Service)(nil)

// Export retrieves the symbol identified by id and serializes it as an imex.Envelope
// stamped with Version. It returns query.ErrNotFound if no symbol exists for id.Key.
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
	env := imex.Envelope{Version: Version, Type: string(s.Type()), Name: sym.Name}
	if err = imex.Encode(&env, sym); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}

// consoleRegion mirrors Region as Console-written files serialize it: camelCase
// keys. Frozen; Console files no longer evolve.
type consoleRegion struct {
	Key         string   `json:"key" msgpack:"key"`
	Name        string   `json:"name" msgpack:"name"`
	Selectors   []string `json:"selectors" msgpack:"selectors"`
	StrokeColor *string  `json:"strokeColor" msgpack:"strokeColor"`
	FillColor   *string  `json:"fillColor" msgpack:"fillColor"`
}

func (r consoleRegion) region() Region {
	return Region{
		Key: r.Key, Name: r.Name, Selectors: r.Selectors,
		StrokeColor: r.StrokeColor, FillColor: r.FillColor,
	}
}

// consoleState mirrors State on the path to consoleRegion.
type consoleState struct {
	Key     string          `json:"key" msgpack:"key"`
	Name    string          `json:"name" msgpack:"name"`
	Regions []consoleRegion `json:"regions" msgpack:"regions"`
}

func (s consoleState) state() State {
	return State{
		Key: s.Key, Name: s.Name,
		Regions: lo.Map(s.Regions, func(r consoleRegion, _ int) Region {
			return r.region()
		}),
	}
}

// consoleSpec mirrors Spec as Console-written files serialize it.
type consoleSpec struct {
	SVG             string            `json:"svg" msgpack:"svg"`
	States          []consoleState    `json:"states" msgpack:"states"`
	Variant         string            `json:"variant" msgpack:"variant"`
	Handles         []Handle          `json:"handles" msgpack:"handles"`
	Scale           float64           `json:"scale" msgpack:"scale"`
	ScaleStroke     bool              `json:"scaleStroke" msgpack:"scaleStroke"`
	PreviewViewport *spatial.Viewport `json:"previewViewport" msgpack:"previewViewport"`
}

func (s consoleSpec) spec() Spec {
	return Spec{
		SVG: s.SVG, Variant: s.Variant, Handles: s.Handles, Scale: s.Scale,
		ScaleStroke: s.ScaleStroke, PreviewViewport: s.PreviewViewport,
		States: lo.Map(s.States, func(st consoleState, _ int) State {
			return st.state()
		}),
	}
}

// consoleSymbol is the decode target for Console-written symbol files, which
// stamp the symbol's old persisted version field ("1") as the envelope version.
type consoleSymbol struct {
	Data consoleSpec `json:"data" msgpack:"data"`
}

func (s *Service) decodeImport(ctx context.Context, env imex.Envelope) (Symbol, error) {
	switch {
	case env.Version > Version:
		return Symbol{}, imex.NewErrUnsupportedVersion(
			string(s.Type()), env.Version, Version,
		)
	case env.Version == Version:
		return imex.Decode[Symbol](ctx, env)
	}
	cs, err := imex.Decode[consoleSymbol](ctx, env)
	if err != nil {
		return Symbol{}, err
	}
	return Symbol{Data: cs.Data.spec()}, nil
}

// Import decodes the envelope into a Symbol and persists it on tx, returning the
// ontology.ID of the newly-created symbol. The exported key is discarded and a fresh
// one is generated so that importing always materializes a new resource. Imported
// symbols are created under the service's permanent symbol group; opts.Project does
// not apply to symbols. Envelopes below Version are Console-written camelCase files;
// an envelope newer than Version is rejected with a path-scoped validation error.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	_ imex.ImportOptions,
) (ontology.ID, error) {
	sym, err := s.decodeImport(ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	sym.Key = uuid.Nil
	// env.Name is the resolved resource name: the body's name when present, or the
	// caller-supplied file name fallback applied by the imex service.
	sym.Name = env.Name
	if err = s.NewWriter(tx).Create(ctx, &sym, s.group.OntologyID()); err != nil {
		return ontology.ID{}, err
	}
	return sym.OntologyID(), nil
}
