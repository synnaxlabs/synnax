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

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/legacy"
)

// specFromConsole lifts the frozen Console v1 export into the current Spec.
func specFromConsole(s legacy.Spec) Spec {
	return Spec{
		SVG: s.SVG, Variant: s.Variant, Scale: s.Scale,
		ScaleStroke: s.ScaleStroke, PreviewViewport: s.PreviewViewport,
		States: lo.Map(s.States, func(st legacy.State, _ int) State {
			return State{
				Key: st.Key, Name: st.Name,
				Regions: lo.Map(st.Regions, func(r legacy.Region, _ int) Region {
					return Region{
						Key: r.Key, Name: r.Name, Selectors: r.Selectors,
						StrokeColor: r.StrokeColor, FillColor: r.FillColor,
					}
				}),
			}
		}),
		Handles: lo.Map(s.Handles, func(h legacy.Handle, _ int) Handle {
			return Handle{
				Key: h.Key, Position: h.Position, Orientation: h.Orientation,
			}
		}),
	}
}

// DecodeImExEnvelope materializes env's body as a current-version Symbol, keyless and
// named after the envelope. An unknown version is a path-scoped validation error.
func DecodeImExEnvelope(ctx context.Context, env imex.Envelope) (Symbol, error) {
	var (
		sym Symbol
		err error
	)
	if env.Version > legacy.LastVersion {
		sym, err = autoDecodeEnvelope(ctx, env)
	} else {
		var d legacy.Data
		if d, err = imex.Decode[legacy.Data](ctx, env); err == nil {
			sym.Data = specFromConsole(d.Spec)
		}
	}
	if err != nil {
		return Symbol{}, err
	}
	// Importing always materializes a new resource, so any key on the wire is dropped
	// and the importer mints a fresh one.
	sym.Key = uuid.Nil
	// The header is the resolved name: the body's name when present, or the file-name
	// fallback the imex service applies. Console-era decodes drop it, so it is stamped
	// here for every path.
	sym.Name = env.Name
	return sym, nil
}
