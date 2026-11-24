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

	"github.com/samber/lo"
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

// Announce attaches a set of the expression and the operations of c for its output to
// the transaction of w on the channel stream, as one batch that goes only to a subject
// who reads c.
func (w Writer) Announce(c Calculation) {
	w.svc.announce(w.tx, w.channelWriter(), c.Output, c, fieldsOf(c, true)...)
}

func (w Writer) channelWriter() channel.Writer {
	return w.channels.NewWriter(w.tx).WithDispatchKey(w.dispatchKey)
}

// Create creates the calculation and gives it a key when it has none. Its output must
// be a virtual channel that is not an index or a built-in channel, and that no other
// record writes. It returns an error wrapping validate.ErrValidation when the
// calculation is not valid or its output breaks that rule, and an error wrapping
// query.ErrNotFound when the output does not exist. It fails as compile does, and sets
// the data type of the output to the type of the expression. A primary that c names
// must be a read, or the error wraps validate.ErrValidation at the path primary. With
// none, Create picks one, which reads stored samples. The commit broadcasts a create
// action that holds the stored calculation.
func (w Writer) Create(ctx context.Context, c *Calculation) error {
	if err := c.Validate(); err != nil {
		return err
	}
	return w.create(ctx, c)
}

// create creates c, which must be valid, as Create does.
func (w Writer) create(ctx context.Context, c *Calculation) error {
	if c.Key == uuid.Nil() {
		c.Key = uuid.New()
	}
	output, err := w.checkOutput(ctx, c.Output)
	if err != nil {
		return err
	}
	if err = w.compile(ctx, output, c); err != nil {
		return err
	}
	if c.Primary, err = w.settlePrimary(
		ctx,
		nil,
		c.Primary,
		c.Primary != nil,
		c.Reads,
	); err != nil {
		return err
	}
	if err := w.table.NewCreate().Exclusive().Entry(c).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.attach(*c, NewCreateAction(CreatePayload{Calculation: *c}))
	w.Announce(*c)
	return nil
}

// attach attaches acts on c as one batch, with the check of who may receive the batch,
// from c as the batch leaves it.
func (w Writer) attach(c Calculation, acts ...Action) {
	seq := w.dispatcher.Attach(w.tx, c.Key, acts...)
	w.svc.checks.Add(w.tx, batchCheck[Key]{
		id:    batchID[Key]{key: c.Key, seq: seq},
		check: w.svc.Readable(c),
	})
}

// CreateOutput creates c with a new output: a virtual channel named name in the group
// with key group, or at the root when group is nil, typed by the expression of c, with
// a free index that timestamps it. It returns the output, then its index. It returns an
// error wrapping validate.ErrValidation when c names an output or the output is not
// valid, with the path of the field at fault, and fails as Create does.
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
	if err := c.Validate(); err != nil {
		return nil, err
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
	if err = w.channelWriter().
		CreateMany(ctx, &created, channel.FreeIndex()); err != nil {
		if b, ok := errors.AsType[channel.BatchError](err); ok {
			err = b.Err
		}
		return nil, err
	}
	c.Output = created[0].Key()
	return created, w.create(ctx, c)
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
// output to the type of the expression. A primary that acts set must be a read. When
// the new expression no longer reads the stored primary, Apply picks another. It
// returns the actions it adds to acts, such as a set of the reads that the new
// expression changed or of the primary that it picked.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	var c Calculation
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Calculation](key)).
		Entry(&c).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	_, computed, err := w.ApplyTo(ctx, c, acts...)
	return computed, err
}

// ApplyTo applies the actions to c, which the caller read in the transaction of w, as
// Apply does. It also returns the calculation that it stores.
func (w Writer) ApplyTo(
	ctx context.Context,
	c Calculation,
	acts ...Action,
) (Calculation, []Action, error) {
	next, err := Reduce(c, acts...)
	if err != nil {
		return Calculation{}, nil, err
	}
	if err = next.Validate(); err != nil {
		return Calculation{}, nil, err
	}
	var output channel.Channel
	if err = w.channels.NewRetrieve().
		Where(channel.MatchKeys(next.Output)).
		Entry(&output).
		Exec(ctx, w.tx); err != nil {
		return Calculation{}, nil, err
	}
	if err = w.compile(ctx, output, &next); err != nil {
		return Calculation{}, nil, err
	}
	requested, set := next.Primary, setsPrimary(acts)
	if next.Primary, err = w.settlePrimary(
		ctx,
		c.Primary,
		requested,
		set,
		next.Reads,
	); err != nil {
		return Calculation{}, nil, err
	}
	if err = w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Calculation](c.Key)).
		Change(func(gorp.Context, Calculation) Calculation { return next }).
		Exec(ctx, w.tx); err != nil {
		return Calculation{}, nil, err
	}
	var computed []Action
	if !slices.Equal(next.Reads, c.Reads) {
		computed = append(computed, NewSetAction(SetPayload{
			Path:  []string{"reads"},
			Value: next.Reads,
		}))
	}
	if !samePrimary(next.Primary, requested) {
		computed = append(computed, primaryAction(next.Primary))
	}
	w.attach(next, slices.Concat(acts, computed)...)
	return next, computed, nil
}

// compile analyzes c as the calculation that writes output. It returns the analysis
// error when the expression does not compile, and an error wrapping
// validate.ErrValidation when the calculations the expression reads lead back to
// output. It sets the data type of output to the type of the expression, and the reads
// of c to the channels the expression reads.
func (w Writer) compile(
	ctx context.Context,
	output channel.Channel,
	c *Calculation,
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
	if err = validate.PathedError(compiler.Check(ctx, w.tx, result, compiler.Config{
		ChannelService: w.channels,
		Channel:        output,
		Calculation:    *c,
	}), "operations"); err != nil {
		return err
	}
	c.Reads = result.Deps
	return nil
}

// checkCycle returns an error wrapping validate.ErrValidation, which names the path,
// when a calculation among the channels with keys deps reads output, directly or
// through other calculations.
func (w Writer) checkCycle(
	ctx context.Context,
	analyzer *compiler.Analyzer,
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

func (w Writer) newAnalyzer() *compiler.Analyzer { return w.svc.newAnalyzer(w.tx) }

// SetReads stores reads, in order and each once, as the reads of c. It keeps the
// primary of c when the reads hold it, and otherwise picks one. It broadcasts a set of
// each field it changes, and writes nothing when it changes no field.
func (w Writer) SetReads(
	ctx context.Context,
	c Calculation,
	reads []channel.Key,
) error {
	reads = slices.Sorted(slices.Values(lo.Uniq(reads)))
	primary, err := w.settlePrimary(ctx, c.Primary, c.Primary, false, reads)
	if err != nil {
		return err
	}
	sameReads := sameKeys(reads, c.Reads)
	if sameReads && samePrimary(primary, c.Primary) {
		return nil
	}
	var next Calculation
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Calculation](c.Key)).
		Change(func(_ gorp.Context, stored Calculation) Calculation {
			stored.Reads, stored.Primary = reads, primary
			next = stored
			return stored
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	var acts []Action
	if !sameReads {
		acts = append(
			acts,
			NewSetAction(SetPayload{Path: []string{"reads"}, Value: reads}),
		)
	}
	if !samePrimary(primary, c.Primary) {
		acts = append(acts, primaryAction(primary))
	}
	w.attach(next, acts...)
	return nil
}

// primaryAction returns the action that sets the primary of a calculation to primary,
// or clears it when primary is nil.
func primaryAction(primary *channel.Key) Action {
	var value any
	if primary != nil {
		value = *primary
	}
	return NewSetAction(SetPayload{Path: []string{"primary"}, Value: value})
}

func samePrimary(a, b *channel.Key) bool {
	return a == b || (a != nil && b != nil && *a == *b)
}

// Delete deletes the calculations with the given keys. A key with no calculation is not
// an error. The output of each one stays as a channel with no producer.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	var deleted []Calculation
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Calculation](keys...)).
		SkipMissingKeys().
		Entries(&deleted).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Calculation](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	// Who read a calculation before the delete learns that it no longer writes its
	// output.
	for _, c := range deleted {
		w.svc.announce(w.tx, w.channelWriter(), c.Output, c, fieldsOf(c, false)...)
	}
	return nil
}

// sameKeys reports whether a and b hold the same keys, in any order and with any
// repeats, without allocating.
func sameKeys(a, b []channel.Key) bool {
	for _, k := range a {
		if !slices.Contains(b, k) {
			return false
		}
	}
	for _, k := range b {
		if !slices.Contains(a, k) {
			return false
		}
	}
	return true
}
