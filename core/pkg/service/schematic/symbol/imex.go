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

// importRegion mirrors Region tolerating both key forms for its multi-word
// fields: server exports write snake_case, Console-written files camelCase, and
// the two share the same envelope version so the importer cannot dispatch on it.
type importRegion struct {
	Key              string   `json:"key" msgpack:"key"`
	Name             string   `json:"name" msgpack:"name"`
	Selectors        []string `json:"selectors" msgpack:"selectors"`
	StrokeColor      *string  `json:"stroke_color" msgpack:"stroke_color"`
	CamelStrokeColor *string  `json:"strokeColor" msgpack:"strokeColor"`
	FillColor        *string  `json:"fill_color" msgpack:"fill_color"`
	CamelFillColor   *string  `json:"fillColor" msgpack:"fillColor"`
}

func (r importRegion) region() Region {
	out := Region{
		Key: r.Key, Name: r.Name, Selectors: r.Selectors,
		StrokeColor: r.StrokeColor, FillColor: r.FillColor,
	}
	if out.StrokeColor == nil {
		out.StrokeColor = r.CamelStrokeColor
	}
	if out.FillColor == nil {
		out.FillColor = r.CamelFillColor
	}
	return out
}

// importState mirrors State on the path to importRegion.
type importState struct {
	Key     string         `json:"key" msgpack:"key"`
	Name    string         `json:"name" msgpack:"name"`
	Regions []importRegion `json:"regions" msgpack:"regions"`
}

func (s importState) state() State {
	out := State{Key: s.Key, Name: s.Name}
	// A nil slice stays nil so a server-exported symbol round-trips byte-equal.
	if s.Regions != nil {
		out.Regions = lo.Map(s.Regions, func(r importRegion, _ int) Region {
			return r.region()
		})
	}
	return out
}

// importSpec mirrors Spec tolerating both key forms for its multi-word fields.
type importSpec struct {
	SVG                  string            `json:"svg" msgpack:"svg"`
	States               []importState     `json:"states" msgpack:"states"`
	Variant              string            `json:"variant" msgpack:"variant"`
	Handles              []Handle          `json:"handles" msgpack:"handles"`
	Scale                float64           `json:"scale" msgpack:"scale"`
	ScaleStroke          bool              `json:"scale_stroke" msgpack:"scale_stroke"`
	CamelScaleStroke     *bool             `json:"scaleStroke" msgpack:"scaleStroke"`
	PreviewViewport      *spatial.Viewport `json:"preview_viewport" msgpack:"preview_viewport"`
	CamelPreviewViewport *spatial.Viewport `json:"previewViewport" msgpack:"previewViewport"`
}

func (s importSpec) spec() Spec {
	out := Spec{
		SVG: s.SVG, Variant: s.Variant, Handles: s.Handles, Scale: s.Scale,
		ScaleStroke: s.ScaleStroke, PreviewViewport: s.PreviewViewport,
	}
	if s.States != nil {
		out.States = lo.Map(s.States, func(st importState, _ int) State {
			return st.state()
		})
	}
	if !out.ScaleStroke && s.CamelScaleStroke != nil {
		out.ScaleStroke = *s.CamelScaleStroke
	}
	if out.PreviewViewport == nil {
		out.PreviewViewport = s.CamelPreviewViewport
	}
	return out
}

// importSymbol is the decode target for symbol envelopes of either provenance.
type importSymbol struct {
	Version uint32     `json:"version" msgpack:"version"`
	Data    importSpec `json:"data" msgpack:"data"`
}

// Import decodes the envelope into a Symbol and persists it on tx, returning the
// ontology.ID of the newly-created symbol. The exported key is discarded and a fresh
// one is generated so that importing always materializes a new resource. Imported
// symbols are created under the service's permanent symbol group; opts.Project does
// not apply to symbols. Console-written files carry camelCase keys and the same
// envelope version as server exports (the symbol's persisted version field doubles as
// the envelope version), so every body decodes through the dual-form import mirror.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	env imex.Envelope,
	_ imex.ImportOptions,
) (ontology.ID, error) {
	if env.Version > Version {
		return ontology.ID{}, imex.NewErrUnsupportedVersion(
			string(s.Type()), env.Version, Version,
		)
	}
	is, err := imex.Decode[importSymbol](ctx, env)
	if err != nil {
		return ontology.ID{}, err
	}
	sym := Symbol{Version: is.Version, Data: is.Data.spec()}
	sym.Key = uuid.Nil
	// env.Name is the resolved resource name: the body's name when present, or the
	// caller-supplied file name fallback applied by the imex service.
	sym.Name = env.Name
	if err = s.NewWriter(tx).Create(ctx, &sym, s.group.OntologyID()); err != nil {
		return ontology.ID{}, err
	}
	return sym.OntologyID(), nil
}
