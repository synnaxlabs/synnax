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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v5 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v5"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/text"
	"github.com/synnaxlabs/x/validate"
)

// lastStateVersion is the final Console state version ("5.0.0" files). A v5
// state parks the plot body under pendingUpload; earlier states embed it inline
// and ride the storage lift's legacy chain.
const lastStateVersion imex.Version = 5

var (
	_ imex.ImportExporter = (*Service)(nil)
	_ imex.Matcher        = (*Service)(nil)
)

// Match reports whether body is a legacy Console line plot state: v0-v4 files persist
// the plot body inline (axes/channels), v5 carries selectedRules/hiddenLines alongside
// an optional pendingUpload. The markers are frozen — they describe historical file
// shapes.
func (s *Service) Match(body map[string]any) bool {
	_, hasAxes := body["axes"]
	_, hasChannels := body["channels"]
	_, hasSelectedRules := body["selectedRules"]
	_, hasHiddenLines := body["hiddenLines"]
	return (hasAxes && hasChannels) || hasSelectedRules || hasHiddenLines
}

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
// fresh one is generated so that importing always materializes a new resource. Line
// plots are project children, so a non-zero opts.Parent must be a project; the plot
// is then created within it exactly as a regular create would be. Envelopes older
// than Version are Console-era files — camelCase typed exports or console states —
// and are lifted forward; an envelope newer than Version is rejected with a
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
	lp, err := s.decodeImport(ctx, env)
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

// stateV5 is the slice of the v5 Console state the importer needs: the plot body
// parked under pendingUpload when a line plot was never uploaded. The tag is
// camelCase because the file was written by the Console.
type stateV5 struct {
	PendingUpload *consoleDocument `json:"pendingUpload" msgpack:"pendingUpload"`
}

// consoleAxis mirrors Axis as Console-written files serialize it: camelCase
// keys. Frozen; Console files no longer evolve.
type consoleAxis struct {
	Key            AxisKey           `json:"key" msgpack:"key"`
	Label          string            `json:"label" msgpack:"label"`
	LabelDirection spatial.Direction `json:"labelDirection" msgpack:"labelDirection"`
	LabelLevel     text.Level        `json:"labelLevel" msgpack:"labelLevel"`
	Bounds         spatial.Bounds    `json:"bounds" msgpack:"bounds"`
	ManualBounds   ManualBounds      `json:"manualBounds" msgpack:"manualBounds"`
	TickSpacing    float64           `json:"tickSpacing" msgpack:"tickSpacing"`
	Type           *TickType         `json:"type,omitempty" msgpack:"type,omitempty"`
}

func (a consoleAxis) axis() Axis {
	return Axis{
		Key: a.Key, Label: a.Label, LabelDirection: a.LabelDirection,
		LabelLevel: a.LabelLevel, Bounds: a.Bounds, ManualBounds: a.ManualBounds,
		TickSpacing: a.TickSpacing, Type: a.Type,
	}
}

// consoleAxes mirrors Axes as Console-written files serialize it.
type consoleAxes struct {
	X1 consoleAxis `json:"x1" msgpack:"x1"`
	X2 consoleAxis `json:"x2" msgpack:"x2"`
	Y1 consoleAxis `json:"y1" msgpack:"y1"`
	Y2 consoleAxis `json:"y2" msgpack:"y2"`
	Y3 consoleAxis `json:"y3" msgpack:"y3"`
	Y4 consoleAxis `json:"y4" msgpack:"y4"`
}

func (a consoleAxes) axes() Axes {
	return Axes{
		X1: a.X1.axis(), X2: a.X2.axis(), Y1: a.Y1.axis(),
		Y2: a.Y2.axis(), Y3: a.Y3.axis(), Y4: a.Y4.axis(),
	}
}

// consoleLine mirrors Line as Console-written files serialize it.
type consoleLine struct {
	Key            string         `json:"key" msgpack:"key"`
	Label          *string        `json:"label,omitempty" msgpack:"label,omitempty"`
	Color          *color.Color   `json:"color,omitempty" msgpack:"color,omitempty"`
	StrokeWidth    float64        `json:"strokeWidth" msgpack:"strokeWidth"`
	Downsample     uint32         `json:"downsample" msgpack:"downsample"`
	DownsampleMode DownsampleMode `json:"downsampleMode" msgpack:"downsampleMode"`
}

func (l consoleLine) line() Line {
	return Line{
		Key: l.Key, Label: l.Label, Color: l.Color, StrokeWidth: l.StrokeWidth,
		Downsample: l.Downsample, DownsampleMode: l.DownsampleMode,
	}
}

// consoleRule mirrors Rule as Console-written files serialize it.
type consoleRule struct {
	Key       string       `json:"key" msgpack:"key"`
	Label     string       `json:"label" msgpack:"label"`
	Color     *color.Color `json:"color,omitempty" msgpack:"color,omitempty"`
	Axis      AxisKey      `json:"axis" msgpack:"axis"`
	LineWidth float64      `json:"lineWidth" msgpack:"lineWidth"`
	LineDash  float64      `json:"lineDash" msgpack:"lineDash"`
	Units     string       `json:"units" msgpack:"units"`
	Position  float64      `json:"position" msgpack:"position"`
}

func (r consoleRule) rule() Rule {
	return Rule{
		Key: r.Key, Label: r.Label, Color: r.Color, Axis: r.Axis,
		LineWidth: r.LineWidth, LineDash: r.LineDash, Units: r.Units,
		Position: r.Position,
	}
}

// consoleDocument mirrors the typed LinePlot body fields as Console-written
// files serialize them: the typed export's top level, and the v5 state's
// pendingUpload.
type consoleDocument struct {
	Title    Title         `json:"title" msgpack:"title"`
	Legend   Legend        `json:"legend" msgpack:"legend"`
	Channels Channels      `json:"channels" msgpack:"channels"`
	Ranges   Ranges        `json:"ranges" msgpack:"ranges"`
	Axes     consoleAxes   `json:"axes" msgpack:"axes"`
	Lines    []consoleLine `json:"lines" msgpack:"lines"`
	Rules    []consoleRule `json:"rules" msgpack:"rules"`
}

// linePlot lifts the Console document into the current LinePlot shape.
func (d consoleDocument) linePlot() LinePlot {
	return LinePlot{
		Title: d.Title, Legend: d.Legend, Channels: d.Channels, Ranges: d.Ranges,
		Axes:  d.Axes.axes(),
		Lines: lo.Map(d.Lines, func(l consoleLine, _ int) Line { return l.line() }),
		Rules: lo.Map(d.Rules, func(r consoleRule, _ int) Rule { return r.rule() }),
	}
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
		doc, err := imex.Decode[consoleDocument](ctx, env)
		if err != nil {
			return LinePlot{}, err
		}
		return doc.linePlot(), nil
	}
	if env.Version == lastStateVersion {
		st, err := imex.Decode[stateV5](ctx, env)
		if err != nil {
			return LinePlot{}, err
		}
		if st.PendingUpload == nil {
			return LinePlot{}, errors.Wrap(
				validate.ErrValidation, "line plot file has no body data",
			)
		}
		return st.PendingUpload.linePlot(), nil
	}
	// v0-v4 console states embed the body inline: ride the storage lift, which
	// dispatches on the version string inside the body.
	body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
	if err != nil {
		return LinePlot{}, err
	}
	return v6.MigrateLinePlot(ctx, v5.LinePlot{Name: env.Name, Data: body})
}
