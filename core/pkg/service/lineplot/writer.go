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
	"slices"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	v10 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v11"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xcolor "github.com/synnaxlabs/x/color"
)

type Writer struct {
	tx         gorp.Tx
	table      *gorp.Table[Key, LinePlot]
	dispatcher actions.Dispatcher[Key, Action]
	color      *color.Service
	ranger     *ranger.Service
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the given line plot, assigning a new key when lp.Key is zero. Each
// line and rule with no color gets a series color, and a repeated range reference is
// dropped. The line plot must belong to a
// project. A line plot in a group belongs to the project that holds the group. Create
// returns an error wrapping validate.ErrValidation when a line plot with lp.Key exists.
func (w Writer) Create(ctx context.Context, lp *LinePlot) error {
	if lp.Key == uuid.Nil() {
		lp.Key = uuid.New()
	}
	if lp.Group != nil {
		proj, err := project.GroupProject(ctx, w.tx, lp.Group, resource.TypeLineplot)
		if err != nil {
			return err
		}
		lp.Project = proj
	}
	// Materialize lines for any channel/range bindings supplied at creation so a plot
	// created with channels and ranges but no lines is fully populated.
	lp.Lines = v10.ReconcileLines(lp.Key, lp.Channels, lp.Lines)
	lp.Ranges = v11.UniqueRanges(lp.Ranges)
	colored, err := w.fillColors(ctx, *lp)
	if err != nil {
		return err
	}
	if *lp, err = Reduce(*lp, colored...); err != nil {
		return err
	}
	if err = lp.Validate(); err != nil {
		return err
	}
	if err := project.CheckMember(lp.Project); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(lp).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(
		w.tx,
		lp.Key,
		NewCreateAction(CreatePayload{LinePlot: *lp}),
	)
	return nil
}

// setProject moves the line plots with keys that are not in the project p into p, and
// attaches a set project action for each one.
func (w Writer) setProject(ctx context.Context, p project.Key, keys ...Key) error {
	var moved []Key
	if err := w.table.NewUpdate().
		Where(gorp.And(
			gorp.MatchKeys[Key, LinePlot](keys...),
			gorp.Match(func(_ gorp.Context, e *LinePlot) (bool, error) {
				return e.Project == nil || *e.Project != p, nil
			}),
		)).
		Change(func(_ gorp.Context, e LinePlot) LinePlot {
			moved = append(moved, e.Key)
			e.Project = &p
			return e
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, k := range moved {
		w.dispatcher.Attach(
			w.tx,
			k,
			NewSetAction(SetPayload{Path: []string{"project"}, Value: p}),
		)
	}
	return nil
}

// CreateMany creates the given line plots. It returns an error wrapping
// validate.ErrValidation when a line plot with one of their keys exists.
func (w Writer) CreateMany(ctx context.Context, plots *[]LinePlot) error {
	for i := range *plots {
		if err := w.Create(ctx, &(*plots)[i]); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().Where(gorp.MatchKeys[Key, LinePlot](keys...)).
		Exec(ctx, w.tx)
}

// Apply applies a sequence of actions to the line plot with the given key and attaches
// them to the transaction of w, so its commit broadcasts them with the dispatch key of
// w. Each line and rule with no color gets a series color. A move into a group also
// moves the line plot into the project that holds the group, and a move into another
// project moves it out of its group. Apply attaches the actions of each such color and
// move after acts and returns them. It returns a validation error, and stores nothing,
// if the actions leave the line plot invalid.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	var added []Action
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, LinePlot](key)).
		ChangeErr(func(_ gorp.Context, prev LinePlot) (LinePlot, error) {
			next, err := Reduce(prev, acts...)
			if err != nil {
				return LinePlot{}, err
			}
			colored, err := w.fillColors(ctx, next)
			if err != nil {
				return LinePlot{}, err
			}
			if next, err = Reduce(next, colored...); err != nil {
				return LinePlot{}, err
			}
			added = colored
			var placed project.Placement
			var moved []Action
			if placed, moved, err = project.Place(
				ctx,
				w.tx,
				resource.TypeLineplot,
				project.Placement{Group: prev.Group, Project: prev.Project},
				project.Placement{Group: next.Group, Project: next.Project},
				newSetAction,
				newRemoveAction,
			); err != nil {
				return LinePlot{}, err
			}
			added = append(added, moved...)
			next.Group, next.Project = placed.Group, placed.Project
			if err = next.Validate(); err != nil {
				return LinePlot{}, err
			}
			return next, project.CheckMember(next.Project)
		}).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	w.dispatcher.Attach(w.tx, key, slices.Concat(acts, added)...)
	return added, nil
}

// fillColors returns an action for each line, rule, and range reference of lp with no
// color. A line or rule gets a series color from pickSeries, and a range reference gets
// a color from pickRangeColors.
func (w Writer) fillColors(ctx context.Context, lp LinePlot) ([]Action, error) {
	lines := lo.Map(lp.Lines, func(l Line, _ int) *color.Ref { return l.Color })
	rules := lo.Map(lp.Rules, func(r Rule, _ int) *color.Ref { return r.Color })
	refs := lo.Map(lp.Ranges, func(r ranger.Ref, _ int) *color.Ref {
		_, c := settings(r)
		return c
	})
	if !slices.Contains(lines, nil) &&
		!slices.Contains(rules, nil) &&
		!slices.Contains(refs, nil) {
		return nil, nil
	}
	series, err := w.color.Series(ctx, w.tx)
	if err != nil || len(series) == 0 {
		return nil, err
	}
	var acts []Action
	for i, pick := range pickSeries(series, lines) {
		if pick != nil {
			acts = append(acts, NewSetLineColorAction(
				SetLineColorPayload{Key: lp.Lines[i].Key, Color: pick},
			))
		}
	}
	for i, pick := range pickSeries(series, rules) {
		if pick != nil {
			acts = append(acts, NewSetRuleColorAction(
				SetRuleColorPayload{Key: lp.Rules[i].Key, Color: pick},
			))
		}
	}
	if !slices.Contains(refs, nil) {
		return acts, nil
	}
	own, err := w.rangeColors(ctx, lp.Ranges, refs)
	if err != nil {
		return nil, err
	}
	named, err := w.namedColors(ctx, series, refs)
	if err != nil {
		return nil, err
	}
	for i, pick := range pickRangeColors(series, refs, own, named) {
		if pick != nil {
			acts = append(acts, NewSetRangeColorAction(
				SetRangeColorPayload{Range: lp.Ranges[i], Color: *pick},
			))
		}
	}
	return acts, nil
}

// rangeColors returns the own color of the saved range of each reference in refs with
// no color, and nil for every other reference. A deleted range has no color.
func (w Writer) rangeColors(
	ctx context.Context,
	refs []ranger.Ref,
	colors []*color.Ref,
) ([]*xcolor.Color, error) {
	var keys []ranger.Key
	for i, r := range refs {
		if k, ok := r.Variant.(ranger.KeyRef); ok && colors[i] == nil {
			keys = append(keys, k.Key)
		}
	}
	own := make([]*xcolor.Color, len(refs))
	if len(keys) == 0 {
		return own, nil
	}
	var ranges []ranger.Range
	if err := w.ranger.NewRetrieve().
		Where(ranger.MatchKeys(keys...)).
		SkipMissingKeys().
		Entries(&ranges).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	byKey := lo.SliceToMap(ranges, func(r ranger.Range) (ranger.Key, *xcolor.Color) {
		return r.Key, r.Color
	})
	for i, r := range refs {
		if k, ok := r.Variant.(ranger.KeyRef); ok && colors[i] == nil {
			own[i] = byKey[k.Key]
		}
	}
	return own, nil
}

// namedColors returns the series colors and the named colors in refs, by key.
func (w Writer) namedColors(
	ctx context.Context,
	series []color.Key,
	refs []*color.Ref,
) (map[color.Key]color.Named, error) {
	keys := slices.Clone(series)
	for _, r := range refs {
		if r == nil {
			continue
		}
		if n, ok := r.Variant.(color.NamedRef); ok && !slices.Contains(keys, n.Key) {
			keys = append(keys, n.Key)
		}
	}
	var named []color.Named
	if err := w.color.NewRetrieve().
		Where(color.MatchKeys(keys...)).
		Entries(&named).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	return lo.SliceToMap(named, func(n color.Named) (color.Key, color.Named) {
		return n.Key, n
	}), nil
}

// minRangeDistance is the least OKLab distance between two colors of a plot's ranges,
// in normal vision and under each color vision deficiency.
const minRangeDistance = 0.05

// apart returns true if every color in as is at least minRangeDistance from every color
// in bs, in normal vision and under each color vision deficiency.
func apart(as, bs []xcolor.Color) bool {
	for _, a := range as {
		for _, b := range bs {
			if xcolor.Distance(a, b) < minRangeDistance {
				return false
			}
			for _, d := range xcolor.Deficiencies {
				sa, sb := xcolor.Simulate(a, d), xcolor.Simulate(b, d)
				if xcolor.Distance(sa, sb) < minRangeDistance {
					return false
				}
			}
		}
	}
	return true
}

// refValues returns the values of r: its fixed value, or both theme values of the named
// color in named. A named color with no dark value has only its light value.
func refValues(r *color.Ref, named map[color.Key]color.Named) []xcolor.Color {
	switch v := r.Variant.(type) {
	case color.FixedRef:
		return []xcolor.Color{v.Value}
	case color.NamedRef:
		n := named[v.Key]
		if n.Dark == nil {
			return []xcolor.Color{n.Light}
		}
		return []xcolor.Color{n.Light, *n.Dark}
	}
	return nil
}

// pickRangeColors returns a color for each empty color in colors, and nil for each set
// one. A pick is the own color of the range when it is apart from every other color in
// colors and earlier picks. Else it is the series color that the fewest of those colors
// are too close to, and a tie goes to the first series color. named holds the value of
// each named color in series and colors.
func pickRangeColors(
	series []color.Key,
	colors []*color.Ref,
	own []*xcolor.Color,
	named map[color.Key]color.Named,
) []*color.Ref {
	values := func(r *color.Ref) []xcolor.Color { return refValues(r, named) }
	var taken []xcolor.Color
	for _, c := range colors {
		if c != nil {
			taken = append(taken, values(c)...)
		}
	}
	picks := make([]*color.Ref, len(colors))
	for i, c := range colors {
		if c == nil && own[i] != nil && apart([]xcolor.Color{*own[i]}, taken) {
			taken = append(taken, *own[i])
			picks[i] = &color.Ref{Variant: color.FixedRef{Value: *own[i]}}
		}
	}
	seriesValues := lo.Map(series, func(k color.Key, _ int) []xcolor.Color {
		return values(&color.Ref{Variant: color.NamedRef{Key: k}})
	})
	uses := make([]int, len(series))
	use := func(vs []xcolor.Color) {
		for i, sv := range seriesValues {
			if !apart(vs, sv) {
				uses[i]++
			}
		}
	}
	for i, c := range colors {
		if c != nil {
			use(values(c))
		} else if picks[i] != nil {
			use(values(picks[i]))
		}
	}
	for i, c := range colors {
		if c != nil || picks[i] != nil {
			continue
		}
		least := 0
		for j := range series {
			if uses[j] < uses[least] {
				least = j
			}
		}
		picks[i] = &color.Ref{Variant: color.NamedRef{Key: series[least]}}
		use(seriesValues[least])
	}
	return picks
}

// pickSeries returns a series color for each empty color in colors, and nil for each
// set one. Each pick is the series color that the fewest colors and earlier picks use,
// and a tie goes to the first series color.
func pickSeries(series []color.Key, colors []*color.Ref) []*color.Ref {
	uses := make(map[color.Key]int, len(series))
	for _, c := range colors {
		if c == nil {
			continue
		}
		if n, ok := c.Variant.(color.NamedRef); ok {
			uses[n.Key]++
		}
	}
	picks := make([]*color.Ref, len(colors))
	for i, c := range colors {
		if c != nil {
			continue
		}
		pick := lo.MinBy(series, func(a, b color.Key) bool { return uses[a] < uses[b] })
		uses[pick]++
		picks[i] = &color.Ref{Variant: color.NamedRef{Key: pick}}
	}
	return picks
}
