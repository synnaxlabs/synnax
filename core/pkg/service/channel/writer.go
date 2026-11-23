// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"fmt"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

const freeIndexNameSuffix = "_time"

// Writer creates, deletes, and renames channels within a single transaction. It changes
// storage channels when
// the transaction commits, before the channel records reach the store, and a storage
// error aborts the commit. A failed commit does not undo the storage changes that ran
// before the failure. Obtain one from Service.NewWriter. A Writer is not safe for
// concurrent use.
type Writer struct {
	svc        *Service
	tx         gorp.Tx
	dispatcher actions.Dispatcher[Key, Action]
}

// Announce attaches acts to the transaction of w for the channel with key without
// applying them, as one batch, and returns the number of the batch. They carry fields
// that the channel does not store to the clients that read them from it.
func (w Writer) Announce(key Key, acts ...Action) uint64 {
	return w.dispatcher.Attach(w.tx, key, acts...)
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

type createOptions struct {
	retrieveIfNameExists bool
	// freeIndex is true when the create gives each virtual, free channel with no index
	// a free index of its own.
	freeIndex bool
	// builtIn is true when Declare creates the channels, which the channel limit does
	// not count.
	builtIn bool
}

// limited reports whether the channel limit counts ch.
func (o createOptions) limited(ch Channel) bool { return !o.builtIn && !ch.Virtual }

// CreateOption configures the behavior of a Create or CreateMany call.
type CreateOption func(*createOptions)

// FreeIndex returns a CreateOption that gives each virtual, free channel with no index
// a free index named for it, adopting a free index of that name that a deleted channel
// left.
func FreeIndex() CreateOption {
	return func(o *createOptions) { o.freeIndex = true }
}

// needsFreeIndex reports whether a create with opts gives ch a free index.
func needsFreeIndex(ch Channel, opts createOptions) bool {
	return ch.LocalIndex == 0 && opts.freeIndex && ch.Virtual &&
		ch.Leaseholder == node.KeyFree
}

// RetrieveIfNameExists returns a CreateOption that populates a channel from the stored
// channel of its name, compared in NFC, instead of creating it. A channel with a group
// takes only the stored channel in that group. A name that several stored channels
// hold returns an AmbiguousNameError.
func RetrieveIfNameExists() CreateOption {
	return func(o *createOptions) { o.retrieveIfNameExists = true }
}

// SetGroup applies a set of the group to each channel with keys, moving it into g, or
// out of its group when g is nil.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	set := NewSetAction(SetPayload{Path: []string{"group"}, Value: g})
	for _, k := range keys {
		if _, err := w.Apply(ctx, k, set); err != nil {
			return err
		}
	}
	return nil
}

// Apply applies the actions to the channel with the given key and attaches them to the
// transaction of w, with any actions it computes, which it returns. A rename reaches
// storage when the transaction commits. Apply returns an error wrapping
// validate.ErrValidation for an invalid action or result, and query.ErrNotFound when no
// channel has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	var prev Channel
	if err := w.svc.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Channel](key)).
		Entry(&prev).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	acts = normalizeNameActions(acts)
	next, err := Reduce(prev, acts...)
	if err != nil {
		return nil, err
	}
	if err = next.Validate(); err != nil {
		return nil, err
	}
	if err = w.checkGroup(ctx, prev, next); err != nil {
		return nil, err
	}
	renamed := next.Name != prev.Name
	if renamed {
		if err = w.checkRename(prev, next.Name, false); err != nil {
			return nil, err
		}
	}
	// One write of the name and the group checks the name in the new group.
	if err = w.svc.table.NewCreate().Entry(&next).Exec(ctx, w.tx); err != nil {
		return nil, withoutPosition(nameTaken(err, 0))
	}
	if renamed {
		w.stageRenames(Keys{key}, []string{next.Name})
	}
	if _, ok := freeIndexOf(next); ok && !sameGroup(prev.Group, next.Group) {
		if err = w.regroupIndex(ctx, key, next); err != nil {
			return nil, err
		}
	}
	var computed []Action
	if next.DataType != prev.DataType {
		computed = append(computed, dataTypeAction(next.DataType))
	}
	w.dispatcher.Attach(w.tx, key, slices.Concat(acts, computed)...)
	return computed, nil
}

// checkGroup checks that next, an edit of prev, sits in a group that holds channels,
// and that a built-in channel stays in its group.
func (w Writer) checkGroup(ctx context.Context, prev, next Channel) error {
	if !sameGroup(prev.Group, next.Group) && w.svc.IsBuiltIn(prev.Key()) {
		return errors.Wrapf(
			validate.ErrValidation,
			"can't move built-in channel %q",
			prev.Name,
		)
	}
	return group.CheckMember(ctx, w.tx, next.Group, resource.TypeChannel, nil)
}

// regroupIndex moves the index of ch, the channel with key, to the group of
// ch unless another channel uses the index.
func (w Writer) regroupIndex(ctx context.Context, key Key, ch Channel) error {
	used, err := w.usedIndexes(ctx, set.New(ch.Index()), set.New(key))
	if err != nil {
		return err
	}
	if used.Contains(ch.Index()) {
		return nil
	}
	return w.SetGroup(ctx, ch.Group, ch.Index())
}

// dataTypeAction returns the action that sets the data type of a channel to dt.
func dataTypeAction(dt telem.DataType) Action {
	return NewSetAction(SetPayload{Path: []string{"data_type"}, Value: dt})
}

// BatchError is the error of the channel at position At of the channels that a Writer
// method takes.
type BatchError struct {
	// At is the position of the channel.
	At int
	// Err is the error of the channel.
	Err error
}

// Error implements error.
func (e BatchError) Error() string { return fmt.Sprintf("%d: %v", e.At, e.Err) }

// Unwrap returns Err.
func (e BatchError) Unwrap() error { return e.Err }

// withoutPosition returns the error of the one channel of a batch that err fails.
func withoutPosition(err error) error {
	if b, ok := errors.AsType[BatchError](err); ok {
		return b.Err
	}
	return err
}

// Create creates a single channel, as CreateMany does.
func (w Writer) Create(ctx context.Context, c *Channel, opts ...CreateOption) error {
	channels := []Channel{*c}
	if err := w.CreateMany(ctx, &channels, opts...); err != nil {
		return withoutPosition(err)
	}
	*c = channels[0]
	return nil
}

// CreateMany creates multiple channels, and the Core assigns the key of each. The error
// of a failed channel is a BatchError at its position in channels.
func (w Writer) CreateMany(
	ctx context.Context, channels *[]Channel, opts ...CreateOption,
) error {
	if len(*channels) == 0 {
		return nil
	}
	var o createOptions
	for _, opt := range opts {
		opt(&o)
	}
	return w.create(ctx, channels, o)
}

// ChangeDataType persists dataType to the existing virtual channel with the given key
// and attaches a set of the data type. It is the write path of the calculation graph,
// which derives the data type of each calculation output. It returns an error wrapping
// validate.ErrValidation when the channel stores its samples, since its storage fixes
// its data type.
func (w Writer) ChangeDataType(
	ctx context.Context, key Key, dataType telem.DataType,
) error {
	if err := w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Channel](key)).
		ChangeErr(func(_ gorp.Context, c Channel) (Channel, error) {
			if !c.Virtual {
				return Channel{}, errors.Wrapf(
					validate.ErrValidation,
					"cannot change the data type of channel %q, which stores its samples",
					c.Name,
				)
			}
			c.DataType = dataType
			return c, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, dataTypeAction(dataType))
	return nil
}

// Delete deletes the channel with the given key, along with its storage. Unless
// allowInternal is true, deleting an internal channel returns an error.
func (w Writer) Delete(ctx context.Context, key Key, allowInternal bool) error {
	return w.DeleteMany(ctx, []Key{key}, allowInternal)
}

// DeleteMany deletes the channels with the given keys, along with their storage. It
// also deletes the free index of each deleted channel when no other channel uses it.
// Unless allowInternal is true, deleting any internal channel returns an error and no
// channels are deleted.
func (w Writer) DeleteMany(ctx context.Context, keys []Key, allowInternal bool) error {
	keys, err := w.withFreeIndexes(ctx, keys)
	if err != nil {
		return err
	}
	return w.delete(ctx, keys, allowInternal)
}

// withFreeIndexes returns keys and the free index of each channel in keys that no
// channel outside keys uses.
func (w Writer) withFreeIndexes(ctx context.Context, keys Keys) (Keys, error) {
	var deleted []Channel
	if err := w.svc.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Channel](keys...)).
		SkipMissingKeys().
		Entries(&deleted).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	deleting := set.New(keys...)
	indexes := set.New[Key]()
	for _, c := range deleted {
		if idx, ok := freeIndexOf(c); ok && !deleting.Contains(idx) {
			indexes.Add(idx)
		}
	}
	used, err := w.usedIndexes(ctx, indexes, deleting)
	if err != nil {
		return nil, err
	}
	return slices.Concat(keys, indexes.Difference(used).Slice()), nil
}

// freeIndexOf returns the free index of ch, which the Core made for ch alone, and false
// when ch has none.
func freeIndexOf(ch Channel) (Key, bool) {
	return ch.Index(), ch.Free() && !ch.IsIndex && ch.LocalIndex != 0
}

// usedIndexes returns the keys in indexes that a channel outside exclude uses as its
// index. The Core keeps such an index where it is when it moves or deletes the channel
// that owns it.
func (w Writer) usedIndexes(
	ctx context.Context,
	indexes set.Set[Key],
	exclude set.Set[Key],
) (set.Set[Key], error) {
	used := set.New[Key]()
	if len(indexes) == 0 {
		return used, nil
	}
	var users []Channel
	if err := w.svc.table.NewRetrieve().
		Where(w.svc.index.Filter(indexes.Slice()...)).
		Where(gorp.Match(func(_ gorp.Context, c *Channel) (bool, error) {
			return !exclude.Contains(c.Key()), nil
		})).
		Entries(&users).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	for _, c := range users {
		used.Add(c.Index())
	}
	return used, nil
}

// Rename renames the channel with the given key to newName. Unless allowInternal is
// true, renaming an internal channel returns an error.
func (w Writer) Rename(
	ctx context.Context, key Key, newName string, allowInternal bool,
) error {
	return withoutPosition(
		w.RenameMany(ctx, []Key{key}, []string{newName}, allowInternal),
	)
}

// RenameMany renames the channels with the given keys to the corresponding entries in
// newNames, which must be parallel to keys, and attaches a rename action for each one.
// Unless allowInternal is true, renaming any internal channel returns an error. The
// error of a name that is not valid is a BatchError at its position in keys.
func (w Writer) RenameMany(
	ctx context.Context, keys []Key, newNames []string, allowInternal bool,
) error {
	newNames = NormalizeNames(newNames)
	if err := w.rename(ctx, keys, newNames, allowInternal); err != nil {
		return err
	}
	for i, k := range keys {
		w.dispatcher.Attach(
			w.tx,
			k,
			NewRenameAction(RenamePayload{Name: newNames[i]}),
		)
	}
	return nil
}

// create orchestrates channel creation: it validates names, resolves existing channels,
// makes free indexes, allocates keys through the distribution allocator, and writes the
// rich channel records to the table.
func (w Writer) create(
	ctx context.Context,
	_channels *[]Channel,
	opts createOptions,
) error {
	channels := *_channels
	if err := w.validateCreate(ctx, channels); err != nil {
		return err
	}
	w.prepareCreate(channels)
	var (
		stored set.Set[int]
		err    error
	)
	// Reset channels that already exist by name to their stored records before index
	// synthesis, so an existing channel keeps its stored index.
	if opts.retrieveIfNameExists {
		if stored, err = w.resetStoredNames(ctx, channels); err != nil {
			return err
		}
	}
	if err = checkCreatedNames(channels, stored); err != nil {
		return err
	}
	// Make a free index for each channel that needs one and has none, new or stored.
	indexChannels, calcNeedingIndex, owners, err := synthesizeIndexes(channels, opts)
	if err != nil {
		return err
	}
	regroup, err := w.adoptIndexes(ctx, indexChannels, opts)
	if err != nil {
		return err
	}
	// An error names the position in the request, so an index channel takes the
	// position of its calculated channel.
	requested := make([]int, len(channels), len(channels)+len(indexChannels))
	for i := range requested {
		requested[i] = i
	}
	requested = append(requested, owners...)
	channels = append(channels, indexChannels...)
	written, err := w.allocateNew(ctx, channels, requested, opts)
	if err != nil {
		return err
	}
	linked, err := w.linkStoredIndexes(ctx, channels, calcNeedingIndex)
	if err != nil {
		return err
	}
	written = append(written, linked...)
	for idx, g := range regroup {
		if err = w.SetGroup(ctx, g, indexChannels[idx].Key()); err != nil {
			return err
		}
	}
	*_channels = channels
	if err = w.attachCreates(ctx, lo.Uniq(written)); err != nil {
		return err
	}
	if len(linked) == 0 || w.svc.cfg.AnnounceLinked == nil {
		return nil
	}
	return w.svc.cfg.AnnounceLinked(ctx, w.tx, w, linked...)
}

// validateCreate validates each channel and its group. checkCreatedNames checks the
// names once the create knows which channels exist.
func (w Writer) validateCreate(ctx context.Context, channels []Channel) error {
	for i, ch := range channels {
		if err := ch.Validate(); err != nil {
			return BatchError{At: i, Err: err}
		}
		if err := validateConcurrency(ch); err != nil {
			return BatchError{At: i, Err: err}
		}
		if err := group.CheckMember(
			ctx,
			w.tx,
			ch.Group,
			resource.TypeChannel,
			nil,
		); err != nil {
			return BatchError{At: i, Err: err}
		}
	}
	return nil
}

// checkCreatedNames puts the name of each channel that the create writes in NFC and
// checks it against the name rule. A channel at a position in stored keeps the name of
// its stored record, which a Core that ran with the old flag may have made. The name
// index refuses a name that another channel holds when the create writes.
func checkCreatedNames(channels []Channel, stored set.Set[int]) error {
	for i := range channels {
		if stored.Contains(i) {
			continue
		}
		channels[i].Name = validate.NormalizeName(channels[i].Name)
		if err := checkNameRule(channels[i].Name); err != nil {
			return BatchError{At: i, Err: validate.PathedError(err, "name")}
		}
	}
	return checkRepeatedNames(Names(channels))
}

// prepareCreate sets the leaseholder of each channel and clears its local key, which
// the Core assigns.
func (w Writer) prepareCreate(channels []Channel) {
	for i, ch := range channels {
		if ch.Leaseholder == 0 {
			channels[i].Leaseholder = w.svc.cfg.HostKey
		}
		channels[i].LocalKey = 0
	}
}

// addPosition adds at to positions, making positions on first use, so a create that
// resets no channel allocates no set.
func addPosition(positions set.Set[int], at int) set.Set[int] {
	if positions == nil {
		positions = make(set.Set[int])
	}
	positions.Add(at)
	return positions
}

// resetStoredNames resets each channel at a position not in reset to the stored channel
// of its name: in its group when it has a group, else in any group. A record that holds
// the name exactly wins over one that holds it after NFC. A name that several channels
// hold returns an AmbiguousNameError.
func (w Writer) resetStoredNames(
	ctx context.Context,
	channels []Channel,
) (set.Set[int], error) {
	var reset set.Set[int]
	names := Names(channels)
	normal := NormalizeNames(names)
	var existing []Channel
	if err := w.svc.newRetrieve().
		Where(MatchNames(lo.Uniq(slices.Concat(names, normal))...)).
		Entries(&existing).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	normalized := make(map[string][]Channel)
	for _, e := range existing {
		n := validate.NormalizeName(e.Name)
		normalized[n] = append(normalized[n], e)
	}
	for i, ch := range channels {
		if reset.Contains(i) {
			continue
		}
		holders := normalized[normal[i]]
		if ch.Group != nil {
			holders = lo.Filter(holders, func(e Channel, _ int) bool {
				return sameGroup(e.Group, ch.Group)
			})
		}
		if exact := lo.Filter(holders, func(e Channel, _ int) bool {
			return e.Name == ch.Name
		}); len(exact) > 0 {
			holders = exact
		}
		switch len(holders) {
		case 0:
		case 1:
			channels[i] = holders[0]
			reset = addPosition(reset, i)
		default:
			return nil, AmbiguousName(ch.Name, KeysFromChannels(holders))
		}
	}
	return reset, nil
}

// synthesizeIndexes returns a new index channel for each channel that needs a free
// index and has none, the positions of those channels in channels, and the position in
// channels of the channel of each new index channel.
func synthesizeIndexes(
	channels []Channel,
	opts createOptions,
) ([]Channel, []int, []int, error) {
	indexChannels := make([]Channel, 0, len(channels))
	calcNeedingIndex := make([]int, 0)
	var owners []int
	batchPositionByName := make(map[string]int, len(channels))
	for i, ch := range channels {
		batchPositionByName[ch.Name] = i
	}
	for i, ch := range channels {
		if !needsFreeIndex(ch, opts) {
			continue
		}
		indexName := ch.Name + freeIndexNameSuffix
		if bi, inBatch := batchPositionByName[indexName]; inBatch {
			// An in-batch channel can only serve as the index if it satisfies the same
			// predicate as an adopted existing index. Adopting a host-leased or
			// persisted index would graft its LocalKey into the channel's
			// free keyspace, producing an index key that resolves to the wrong channel.
			if !isFreeIndex(channels[bi]) {
				return nil, nil, nil, errors.Wrapf(
					validate.ErrValidation,
					"channel %q in the same request cannot serve as the index for calculated channel %q",
					indexName,
					ch.Name,
				)
			}
			calcNeedingIndex = append(calcNeedingIndex, i)
			continue
		}
		indexChannels = append(indexChannels, Channel{
			Name:        indexName,
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Virtual:     true,
			Leaseholder: node.KeyFree,
			Group:       ch.Group,
		})
		calcNeedingIndex = append(calcNeedingIndex, i)
		owners = append(owners, i)
	}
	return indexChannels, calcNeedingIndex, owners, nil
}

// isFreeIndex reports whether ch can serve as the index of a free channel.
func isFreeIndex(ch Channel) bool {
	return ch.IsIndex && ch.Virtual && ch.Leaseholder == node.KeyFree &&
		ch.DataType == telem.TimestampT
}

// adoptIndexes resolves the names of indexChannels against existing channels. A
// compatible index, such as one left behind by a deleted free channel, replaces
// its entry in indexChannels instead of a duplicate name. An incompatible channel is a
// validation error. adoptIndexes returns the position in indexChannels of each adopted
// index to move to the group of its new free channel.
func (w Writer) adoptIndexes(
	ctx context.Context,
	indexChannels []Channel,
	opts createOptions,
) (map[int]*group.Key, error) {
	regroup := make(map[int]*group.Key)
	if len(indexChannels) == 0 {
		return regroup, nil
	}
	indexNames := Names(indexChannels)
	var existing []Channel
	if err := w.svc.newRetrieve().
		Where(MatchNames(indexNames...)).
		Entries(&existing).
		Exec(ctx, w.tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	namePositions := positions(indexNames)
	for _, ex := range existing {
		idx := namePositions[ex.Name][0]
		if isFreeIndex(ex) {
			if g := indexChannels[idx].Group; !sameGroup(ex.Group, g) {
				regroup[idx] = g
			}
			indexChannels[idx] = ex
			continue
		}
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"channel %q already exists and cannot serve as the index for a calculated channel",
			ex.Name,
		)
	}
	adopted := set.New[Key]()
	for idx := range regroup {
		adopted.Add(indexChannels[idx].Key())
	}
	used, err := w.usedIndexes(ctx, adopted, nil)
	if err != nil {
		return nil, err
	}
	for idx, g := range regroup {
		if used.Contains(indexChannels[idx].Key()) {
			delete(regroup, idx)
			continue
		}
		indexChannels[idx].Group = g
	}
	return regroup, nil
}

// allocateNew allocates local keys for the channels without one through the
// distribution allocator, writes them, and returns their keys. An error names the
// position in requested of the channel that a name index refuses.
func (w Writer) allocateNew(
	ctx context.Context,
	channels []Channel,
	requested []int,
	opts createOptions,
) (Keys, error) {
	newIndices := make([]int, 0, len(channels))
	minimal := make([]channel.Channel, 0, len(channels))
	for i, ch := range channels {
		if ch.LocalKey == 0 {
			newIndices = append(newIndices, i)
			minimal = append(minimal, ch.Distribution())
		}
	}
	if err := w.allocateAndWrite(
		ctx,
		channels,
		requested,
		newIndices,
		minimal,
		opts,
	); err != nil {
		return nil, err
	}
	written := make(Keys, 0, len(newIndices))
	for _, i := range newIndices {
		written = append(written, channels[i].Key())
	}
	return written, nil
}

// linkStoredIndexes persists the index of each stored free channel at the
// positions calcNeedingIndex whose index channel was just created. It returns the keys
// of the channels it updates.
func (w Writer) linkStoredIndexes(
	ctx context.Context,
	channels []Channel,
	calcNeedingIndex []int,
) (Keys, error) {
	var linked Keys
	for _, idx := range calcNeedingIndex {
		ch := channels[idx]
		if ch.LocalKey == 0 || ch.LocalIndex == 0 {
			continue
		}
		exists, err := w.svc.newRetrieve().Where(MatchKeys(ch.Key())).Exists(ctx, w.tx)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		if err = w.svc.table.NewUpdate().
			Where(gorp.MatchKeys[Key, Channel](ch.Key())).
			Change(func(_ gorp.Context, c Channel) Channel {
				c.LocalIndex = ch.LocalIndex
				return c
			}).
			Exec(ctx, w.tx); err != nil {
			return nil, err
		}
		linked = append(linked, ch.Key())
	}
	return linked, nil
}

func validateConcurrency(ch Channel) error {
	switch ch.Concurrency {
	case control.ConcurrencyExclusive:
		return nil
	case control.ConcurrencyShared:
		if ch.Virtual {
			return nil
		}
		return validate.PathedError(
			errors.Wrap(
				validate.ErrValidation,
				"only virtual channels can have shared concurrency",
			),
			"concurrency",
		)
	default:
		return validate.PathedError(
			errors.Wrapf(
				validate.ErrValidation,
				"%d is not a valid concurrency",
				ch.Concurrency,
			),
			"concurrency",
		)
	}
}

func sameGroup(a, b *group.Key) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// attachCreates attaches a create action that holds the stored channel for each key.
func (w Writer) attachCreates(ctx context.Context, keys Keys) error {
	if len(keys) == 0 {
		return nil
	}
	var stored []Channel
	if err := w.svc.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Channel](keys...)).
		Entries(&stored).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, c := range stored {
		w.dispatcher.Attach(w.tx, c.Key(), NewCreateAction(CreatePayload{
			Channel: Created{Channel: c},
		}))
	}
	return nil
}

// allocateAndWrite enforces the channel limit, then allocates keys for the new channels
// and writes them to the table. When the transaction commits, it creates their storage
// channels and records them in the external non-virtual set.
func (w Writer) allocateAndWrite(
	ctx context.Context,
	channels []Channel,
	requested []int,
	newIndices []int,
	minimal []channel.Channel,
	opts createOptions,
) error {
	if err := w.checkLimit(channels, newIndices, opts); err != nil {
		return err
	}
	toCreate := make([]Channel, 0, len(newIndices))
	var allocated []channel.Channel
	if len(minimal) != 0 {
		var err error
		if allocated, err = w.svc.cfg.Channel.Allocate(ctx, minimal); err != nil {
			return err
		}
		for j, i := range newIndices {
			channels[i].LocalKey = allocated[j].LocalKey
			channels[i].LocalIndex = allocated[j].LocalIndex
			toCreate = append(toCreate, channels[i])
		}
	}

	linkFreeIndexes(channels, toCreate, opts)

	externalCreatedKeys := make(Keys, 0, len(toCreate))
	for _, ch := range toCreate {
		if opts.limited(ch) {
			externalCreatedKeys = append(externalCreatedKeys, ch.Key())
		}
	}
	// A data channel's index must exist before the data channel is written.
	slices.SortStableFunc(toCreate, indexesFirst)
	// A refused name leaves a gap in the allocated keys, which are never reused.
	if err := w.svc.table.NewCreate().
		Exclusive().
		Entries(&toCreate).
		Exec(ctx, w.tx); err != nil {
		at, name := -1, takenName(err)
		for _, i := range newIndices {
			if channels[i].Name == name {
				at = requested[i]
			}
		}
		return nameTaken(err, at)
	}
	gorp.BeforeCommit(w.tx, func(ctx context.Context) error {
		stored := channel.ToStorage(allocated)
		if err := w.svc.cfg.TS.CreateChannel(ctx, stored...); err != nil {
			return err
		}
		w.svc.mu.Lock()
		w.svc.mu.externalNonVirtualSet.Insert(externalCreatedKeys...)
		w.svc.mu.Unlock()
		return nil
	})
	return nil
}

// checkLimit checks that the channels at newIndices in channels that the limit counts
// fit the channel limit.
func (w Writer) checkLimit(
	channels []Channel,
	newIndices []int,
	opts createOptions,
) error {
	externalNewCount := 0
	for _, i := range newIndices {
		if opts.limited(channels[i]) {
			externalNewCount++
		}
	}
	w.svc.mu.RLock()
	count := w.svc.mu.externalNonVirtualSet.Size()
	w.svc.mu.RUnlock()
	return w.svc.cfg.Limit(types.Uint20(int(count) + externalNewCount))
}

// linkFreeIndexes links each channel in toCreate that needs a free index to its index
// channel in channels, by setting the LocalIndex of the channel to the LocalKey of the
// index.
func linkFreeIndexes(channels, toCreate []Channel, opts createOptions) {
	indexKeyByName := make(map[string]LocalKey, len(channels))
	for _, ch := range channels {
		if ch.IsIndex {
			indexKeyByName[ch.Name] = ch.LocalKey
		}
	}
	link := func(chs []Channel) {
		for i := range chs {
			if !needsFreeIndex(chs[i], opts) {
				continue
			}
			if k, ok := indexKeyByName[chs[i].Name+freeIndexNameSuffix]; ok {
				chs[i].LocalIndex = k
			}
		}
	}
	link(toCreate)
	link(channels)
}

// indexesFirst orders an index channel before a channel that is not an index.
func indexesFirst(a, b Channel) int {
	switch {
	case a.IsIndex == b.IsIndex:
		return 0
	case a.IsIndex:
		return -1
	default:
		return 1
	}
}

// checkNameRule returns a validation error when name breaks the name rule
// (validate.IsPathSegment). It leaves an empty name to Channel.Validate.
func checkNameRule(name string) error {
	if name == "" || validate.IsPathSegment(name) {
		return nil
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"channel name '%s' must hold a visible character, and no '.' or invisible "+
			"character",
		name,
	)
}

// checkRepeatedNames rejects a request that names one channel name twice.
func checkRepeatedNames(names []string) error {
	namesSeen := make(set.Set[string], len(names))
	for i, name := range names {
		if namesSeen.Contains(name) {
			return BatchError{At: i, Err: validate.PathedError(
				errors.Wrapf(
					validate.ErrValidation,
					"duplicate channel name '%s' in request",
					name,
				),
				"name",
			)}
		}
		namesSeen.Add(name)
	}
	return nil
}

// takenName returns the name of a clash of the name index or of sibling names, or ""
// when err is not one.
func takenName(err error) string {
	var clash gorp.UniqueError
	if !errors.As(err, &clash) {
		return ""
	}
	switch v := clash.Value.(type) {
	case string:
		return v
	case group.Sibling:
		return v.Name
	}
	return ""
}

// nameTaken returns the error of the channel at position i when err is a clash of the
// name index or of sibling names, and err itself otherwise.
func nameTaken(err error, i int) error {
	var clash gorp.UniqueError
	if !errors.As(err, &clash) {
		return err
	}
	if clash.Space != nameSpace {
		return BatchError{At: i, Err: group.SiblingError(err)}
	}
	return BatchError{At: i, Err: validate.PathedError(
		validate.WithCause(err, "channel with name '%v' already exists", clash.Value),
		"name",
	)}
}

func (w Writer) delete(ctx context.Context, keys Keys, allowInternal bool) error {
	if err := w.svc.table.NewDelete().
		Where(gorp.MatchKeys[Key, Channel](keys...)).
		Guard(func(_ gorp.Context, c Channel) error {
			if !allowInternal && w.svc.IsBuiltIn(c.Key()) {
				return errors.Wrapf(
					validate.ErrValidation,
					"can't delete built-in channel %q",
					c.Name,
				)
			}
			return nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	gorp.BeforeCommit(w.tx, func(ctx context.Context) error {
		if err := w.svc.deleteStorage(keys); err != nil {
			return err
		}
		w.svc.mu.Lock()
		w.svc.mu.externalNonVirtualSet.Remove(keys...)
		w.svc.mu.Unlock()
		return nil
	})
	return nil
}

// rename renames the channels with keys to names, which must be in NFC.
func (w Writer) rename(
	ctx context.Context,
	keys Keys,
	names []string,
	allowInternal bool,
) error {
	if len(keys) != len(names) {
		return errors.Wrap(
			validate.ErrValidation,
			"keys and names must be the same length",
		)
	}
	if err := checkRepeatedNames(names); err != nil {
		return err
	}
	renameMap := make(map[Key]string, len(keys))
	for i, key := range keys {
		renameMap[key] = names[i]
	}
	if err := w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Channel](keys...)).
		ChangeErr(func(_ gorp.Context, c Channel) (Channel, error) {
			next := renameMap[c.Key()]
			if err := w.checkRename(c, next, allowInternal); err != nil {
				return Channel{}, BatchError{At: slices.Index(keys, c.Key()), Err: err}
			}
			c.Name = next
			if err := c.Validate(); err != nil {
				return Channel{}, err
			}
			return c, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return nameTaken(err, slices.Index(names, takenName(err)))
	}
	w.stageRenames(keys, names)
	return nil
}

// checkRename returns a validation error when c cannot take the name next. A stored
// name that the rule refuses stays writable while a rename keeps it.
func (w Writer) checkRename(c Channel, next string, allowInternal bool) error {
	if !allowInternal && w.svc.IsBuiltIn(c.Key()) {
		return errors.Wrapf(
			validate.ErrValidation,
			"can't rename built-in channel %q",
			c.Name,
		)
	}
	if next == c.Name {
		return nil
	}
	return validate.PathedError(checkNameRule(next), "name")
}

// stageRenames records that the channels with keys take names in storage when the
// transaction of w commits.
func (w Writer) stageRenames(keys Keys, names []string) {
	for i, key := range keys {
		w.svc.renames.Add(w.tx, rename{key: key, name: names[i]})
	}
	if len(keys) > 0 {
		count := len(w.svc.renames.Pending(w.tx))
		gorp.BeforeCommit(w.tx, func(ctx context.Context) error {
			return w.renameStored(ctx, count)
		})
	}
}

// rename is a new name for a channel.
type rename struct {
	key  Key
	name string
}

// renameStored sends every rename in the transaction of w to storage in one call when
// count is the number of renames in the transaction. Only the hook of the last rename
// sends. It runs after the storage creates registered before that rename.
func (w Writer) renameStored(ctx context.Context, count int) error {
	renames := w.svc.renames.Pending(w.tx)
	if len(renames) != count {
		return nil
	}
	names := make(map[Key]string, len(renames))
	for _, r := range renames {
		names[r.key] = r.name
	}
	return w.svc.renameStorage(ctx, names)
}

// positions maps each value in vals to the positions that hold it, in ascending order.
func positions[T comparable](vals []T) map[T][]int {
	pos := make(map[T][]int, len(vals))
	for i, v := range vals {
		pos[v] = append(pos[v], i)
	}
	return pos
}
