// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"
	"slices"
	"strings"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, changes, and deletes calculations in a transaction.
type Writer struct {
	svc *Service
	tx  gorp.Tx
	// dispatchKey is stamped onto the broadcasts of the channel edits the writer makes.
	dispatchKey string
	table       *gorp.Table[Key, Calculation]
	indexes     indexes
	channels    *channel.Service
	dispatcher  actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	w.dispatchKey = dispatchKey
	return w
}

// Create creates the calculation and gives it a key when it has none. Its output must
// be a virtual channel that is not an index or a built-in channel, and that no other
// record writes. It returns an error wrapping validate.ErrValidation when the
// calculation is not valid or its output breaks that rule, and an error wrapping
// query.ErrNotFound when the output does not exist. It fails as compile does, and sets
// the data type of the output to the type of the expression. The commit broadcasts a
// create action that holds the stored calculation.
func (w Writer) Create(ctx context.Context, c *Calculation) error {
	if c.Key == uuid.Nil() {
		c.Key = uuid.New()
	}
	if err := c.Validate(); err != nil {
		return err
	}
	output, err := w.checkOutput(ctx, c.Output)
	if err != nil {
		return err
	}
	if err = w.compile(ctx, output, *c); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(c).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, c.Key, NewCreateAction(CreatePayload{Calculation: *c}))
	return nil
}

// CreateOutput creates c with a new output: a virtual channel named name in the group
// with key group, or at the root when group is nil, typed by the expression of c, with
// a
// free index that timestamps it. It returns the output, then its index. It returns an
// error wrapping validate.ErrValidation when c names an output, and fails as Create
// does.
func (w Writer) CreateOutput(
	ctx context.Context,
	c *Calculation,
	name string,
	group *group.Key,
) ([]channel.Channel, error) {
	if c.Output != 0 {
		return nil, validate.PathedError(errors.Wrap(
			validate.ErrValidation,
			"a calculation that makes its output names none",
		), "output")
	}
	output := channel.Channel{
		Name:        name,
		Virtual:     true,
		Leaseholder: node.KeyFree,
		Group:       group,
	}
	result, err := w.newAnalyzer().AnalyzeExpression(
		ctx,
		output,
		c.Expression,
		compiler.HasDerivative(c.Operations),
	)
	if err != nil {
		return nil, validate.PathedError(err, "expression")
	}
	output.DataType = result.ChanDataType
	created := []channel.Channel{output}
	if err = w.channels.NewWriter(w.tx).
		WithDispatchKey(w.dispatchKey).
		CreateMany(ctx, &created, channel.FreeIndex()); err != nil {
		return nil, err
	}
	c.Output = created[0].Key()
	return created, w.Create(ctx, c)
}

// checkOutput returns the channel with key. It fails when the channel does not exist,
// stores its samples, is an index or a built-in channel, or has a record that writes
// it.
func (w Writer) checkOutput(
	ctx context.Context,
	key channel.Key,
) (channel.Channel, error) {
	var ch channel.Channel
	if err := w.channels.NewRetrieve().
		Where(channel.MatchKeys(key)).
		Entry(&ch).
		Exec(ctx, w.tx); err != nil {
		return ch, err
	}
	switch {
	case !ch.Virtual:
		return ch, outputError(
			"channel %s stores its samples, so no calculation writes it",
			ch,
		)
	case ch.IsIndex:
		return ch, outputError(
			"channel %s is an index, so no calculation writes it",
			ch,
		)
	case w.channels.IsBuiltIn(key):
		return ch, outputError(
			"the Core writes channel %s, so no calculation writes it",
			ch,
		)
	}
	uses, err := w.channels.Uses(ctx, w.tx, []channel.Key{key})
	if err != nil {
		return ch, err
	}
	for _, u := range uses[0] {
		if u.Direction == channel.DirectionWrite {
			return ch, outputError("another record already writes channel %s", ch)
		}
	}
	return ch, nil
}

func outputError(format string, ch channel.Channel) error {
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, format, ch.Name),
		"output",
	)
}

// Apply applies the actions to the calculation with the given key, and the commit
// broadcasts them. It returns an error wrapping validate.ErrValidation when an action
// is a create or the result is not valid, and an error wrapping query.ErrNotFound when
// no calculation has the key. It fails as compile does, and sets the data type of the
// output to the type of the expression.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	var c Calculation
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Calculation](key)).
		Entry(&c).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	next, err := Reduce(c, acts...)
	if err != nil {
		return err
	}
	if err = next.Validate(); err != nil {
		return err
	}
	var output channel.Channel
	if err = w.channels.NewRetrieve().
		Where(channel.MatchKeys(next.Output)).
		Entry(&output).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if err = w.compile(ctx, output, next); err != nil {
		return err
	}
	if err = w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Calculation](key)).
		Change(func(gorp.Context, Calculation) Calculation { return next }).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// compile analyzes c as the calculation that writes output. It returns the analysis
// error when the expression does not compile, and an error wrapping
// validate.ErrValidation when the primary of c is not a channel the expression reads or
// the calculations the expression reads lead back to output. It sets the data type of
// output to the type of the expression.
func (w Writer) compile(
	ctx context.Context,
	output channel.Channel,
	c Calculation,
) error {
	analyzer := w.newAnalyzer()
	result, err := analyzer.AnalyzeExpression(
		ctx,
		output,
		c.Expression,
		compiler.HasDerivative(c.Operations),
	)
	if err != nil {
		return validate.PathedError(err, "expression")
	}
	if c.Primary != nil && !slices.Contains(result.Deps, *c.Primary) {
		return validate.PathedError(errors.Wrapf(
			validate.ErrValidation,
			"channel %s is not a channel that the expression reads",
			*c.Primary,
		), "primary")
	}
	reads := slices.Concat(result.Deps, resets(c.Operations))
	if err = w.checkCycle(ctx, analyzer, output, reads); err != nil {
		return err
	}
	if output.DataType != result.ChanDataType {
		if err = w.channels.NewWriter(w.tx).
			ChangeDataType(ctx, output.Key(), result.ChanDataType); err != nil {
			return err
		}
		output.DataType = result.ChanDataType
	}
	return validate.PathedError(compiler.Check(ctx, w.tx, compiler.Config{
		ChannelService: w.channels,
		Channel:        output,
		Calculation:    c,
	}), "operations")
}

// checkCycle returns an error wrapping validate.ErrValidation, which names the path,
// when
// a calculation among the channels with keys deps reads output, directly or through
// other calculations.
func (w Writer) checkCycle(
	ctx context.Context,
	analyzer *channel.CalculationAnalyzer,
	output channel.Channel,
	deps channel.Keys,
) error {
	type step struct {
		key  channel.Key
		path channel.Keys
	}
	queue := make([]step, 0, len(deps))
	for _, d := range deps {
		queue = append(queue, step{key: d, path: channel.Keys{output.Key(), d}})
	}
	visited := set.New[channel.Key]()
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		if s.key == output.Key() {
			return w.cycleError(ctx, s.path)
		}
		if visited.Contains(s.key) {
			continue
		}
		visited.Add(s.key)
		level, err := w.svc.pendingOf(ctx, w.tx, channel.Keys{s.key})
		if err != nil {
			return err
		}
		for _, p := range level {
			reads, err := calculationReads(ctx, analyzer, p)
			if err != nil {
				return err
			}
			for _, r := range reads {
				queue = append(
					queue,
					step{key: r, path: append(slices.Clone(s.path), r)},
				)
			}
		}
	}
	return nil
}

// cycleError returns an error wrapping validate.ErrValidation that names the channels
// of path, a cycle of calculations that read each other.
func (w Writer) cycleError(ctx context.Context, path channel.Keys) error {
	var channels []channel.Channel
	if err := w.channels.NewRetrieve().
		Where(channel.MatchKeys(path.Unique()...)).
		Entries(&channels).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	names := make(map[channel.Key]string, len(channels))
	for _, ch := range channels {
		names[ch.Key()] = ch.Name
	}
	steps := make([]string, len(path))
	for i, k := range path {
		steps[i] = names[k]
	}
	return validate.PathedError(errors.Wrapf(
		validate.ErrValidation,
		"the calculations of %s read each other",
		strings.Join(steps, " -> "),
	), "expression")
}

func (w Writer) newAnalyzer() *channel.CalculationAnalyzer { return w.svc.newAnalyzer(w.tx) }

// Delete deletes the calculations with the given keys. A key with no calculation is not
// an error. The output of each one stays as a channel with no producer.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Calculation](keys...)).
		Exec(ctx, w.tx)
}

// storedChannels returns the stored channels that have the name or key of a channel in
// channels.
func (w Writer) storedChannels(
	ctx context.Context,
	channels []channel.Channel,
) ([]channel.Channel, error) {
	var byName []channel.Channel
	if err := w.channels.NewRetrieve().
		Where(channel.MatchNames(channel.Names(channels)...)).
		Entries(&byName).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	var keys []channel.Key
	for _, ch := range channels {
		if ch.LocalKey != 0 {
			keys = append(keys, ch.Key())
		}
	}
	if len(keys) == 0 {
		return byName, nil
	}
	var byKey []channel.Channel
	if err := w.channels.NewRetrieve().
		Where(channel.MatchKeys(keys...)).
		SkipMissingKeys().
		Entries(&byKey).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	return slices.Concat(byName, byKey), nil
}
