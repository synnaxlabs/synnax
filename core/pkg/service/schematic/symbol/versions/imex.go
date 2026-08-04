// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/spatial"
)

// consoleRegion mirrors Region as Console-written files serialize it: camelCase
// keys. Frozen; Console files no longer evolve.
type consoleRegion struct {
	Key         string   `json:"key"         msgpack:"key"`
	Name        string   `json:"name"        msgpack:"name"`
	Selectors   []string `json:"selectors"   msgpack:"selectors"`
	StrokeColor *string  `json:"strokeColor" msgpack:"strokeColor"`
	FillColor   *string  `json:"fillColor"   msgpack:"fillColor"`
}

func (r consoleRegion) region() Region {
	return Region{
		Key: r.Key, Name: r.Name, Selectors: r.Selectors,
		StrokeColor: r.StrokeColor, FillColor: r.FillColor,
	}
}

// consoleState mirrors State on the path to consoleRegion.
type consoleState struct {
	Key     string          `json:"key"     msgpack:"key"`
	Name    string          `json:"name"    msgpack:"name"`
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
	SVG             string            `json:"svg"             msgpack:"svg"`
	States          []consoleState    `json:"states"          msgpack:"states"`
	Variant         string            `json:"variant"         msgpack:"variant"`
	Handles         []Handle          `json:"handles"         msgpack:"handles"`
	Scale           float64           `json:"scale"           msgpack:"scale"`
	ScaleStroke     bool              `json:"scaleStroke"     msgpack:"scaleStroke"`
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

// DecodeImport materializes the envelope's body as a current-version Symbol.
// Envelopes stamped at or above Floor decode through the generated migration
// chain; envelopes below Floor are Console-written camelCase files. An envelope
// newer than Latest is rejected with a path-scoped validation error.
func DecodeImport(ctx context.Context, env imex.Envelope) (Symbol, error) {
	if env.Version >= Floor {
		return decodeMigrate(ctx, env)
	}
	cs, err := imex.Decode[consoleSymbol](ctx, env)
	if err != nil {
		return Symbol{}, err
	}
	return Symbol{Data: cs.Data.spec()}, nil
}
