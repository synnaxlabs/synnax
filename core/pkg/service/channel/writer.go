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
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

const calculatedIndexNameSuffix = "_time"

// Writer creates, deletes, and renames channels within a single transaction. It infers
// DataTypes for calculated channels before persisting. It changes storage channels when
// the transaction commits, before the channel records reach the store, and a storage
// error aborts the commit. A failed commit does not undo the storage changes that ran
// before the failure. Obtain one from Service.NewWriter. A Writer is not safe for
// concurrent use.
type Writer struct {
	svc        *Service
	tx         gorp.Tx
	analyzer   *CalculationAnalyzer
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

type createOptions struct {
	retrieveIfNameExists                        bool
	overwriteIfNameExistsAndDifferentProperties bool
}

// CreateOption configures the behavior of a Create or CreateMany call.
type CreateOption func(*createOptions)

// RetrieveIfNameExists returns a CreateOption that, when a channel with the same name
// or key already exists, populates the provided channel from the existing record
// instead of returning an error.
func RetrieveIfNameExists() CreateOption {
	return func(o *createOptions) { o.retrieveIfNameExists = true }
}

// OverwriteIfNameExistsAndDifferentProperties returns a CreateOption that overwrites an
// existing channel of the same name when its properties differ from the channel being
// created.
func OverwriteIfNameExistsAndDifferentProperties() CreateOption {
	return func(o *createOptions) {
		o.overwriteIfNameExistsAndDifferentProperties = true
	}
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
// transaction of w. An edit of the expression or operations infers the data type again,
// and Apply attaches and returns a set of a changed data type. A rename reaches storage
// when the transaction commits. Apply returns an error wrapping validate.ErrValidation
// for an invalid action or result, and query.ErrNotFound when no channel has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) ([]Action, error) {
	var prev Channel
	if err := w.svc.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Channel](key)).
		Entry(&prev).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	next, err := Reduce(prev, acts...)
	if err != nil {
		return nil, err
	}
	if err = next.Validate(); err != nil {
		return nil, err
	}
	if err = group.CheckMember(
		ctx,
		w.tx,
		next.Group,
		resource.TypeChannel,
		nil,
	); err != nil {
		return nil, err
	}
	if next.Expression != prev.Expression ||
		!slices.Equal(next.Operations, prev.Operations) {
		if !prev.IsCalculated() || !next.IsCalculated() {
			return nil, errors.Wrap(
				validate.ErrValidation,
				"only a calculated channel can change its expression or operations, "+
					"and its expression cannot be empty",
			)
		}
		result, err := w.analyzer.Analyze(ctx, next)
		if err != nil {
			return nil, err
		}
		next.DataType = result.ChanDataType
	}
	if next.Name != prev.Name {
		if err = w.rename(ctx, Keys{key}, []string{next.Name}, false); err != nil {
			return nil, err
		}
	}
	if err = w.svc.table.NewCreate().Entry(&next).Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	// Opening the service clears the index of a calculated channel when it is missing.
	if next.IsCalculated() && next.LocalIndex != 0 &&
		!sameGroup(prev.Group, next.Group) {
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

// regroupIndex moves the index of ch, the calculated channel with key, to the group of
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

// Create creates a single channel, inferring the DataType if it is calculated. It
// returns an error wrapping validate.ErrValidation when c is calculated and a channel
// with its key exists.
func (w Writer) Create(ctx context.Context, c *Channel, opts ...CreateOption) error {
	channels := []Channel{*c}
	if err := w.CreateMany(ctx, &channels, opts...); err != nil {
		return err
	}
	*c = channels[0]
	return nil
}

// CreateMany creates multiple channels, inferring and assigning DataTypes for any
// calculated channels in the batch by analyzing their expressions. Channels within the
// batch may reference each other by name. A calculated channel whose expression fails
// to analyze (invalid syntax or unresolved dependencies) aborts the entire call with
// the analysis error, so callers get fail-fast validation. The Core assigns the key of
// a channel that is not calculated. CreateMany returns an error wrapping
// validate.ErrValidation when a calculated channel's key exists.
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
	for i, ch := range *channels {
		if !ch.IsCalculated() {
			continue
		}
		result, err := w.analyzer.Analyze(ctx, ch)
		if err != nil {
			return err
		}
		(*channels)[i].DataType = result.ChanDataType
	}
	return w.create(ctx, channels, o)
}

// ChangeDataType persists dataType to the already-existing channel with the given key,
// without analyzing its expression. It is the calculation graph's write path for the
// calculated-channel DataTypes it derives via a cross-channel fixpoint: the graph is
// their source of truth, so re-deriving them here — as CreateMany would — is both
// redundant and, for interdependent channels analyzed in one pass against pre-update
// storage, incorrect. It returns validate.Error if the target channel is not
// calculated, since a non-calculated channel's DataType is fixed by its persistent
// storage. It attaches a set of the data type.
func (w Writer) ChangeDataType(
	ctx context.Context, key Key, dataType telem.DataType,
) error {
	if err := w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Channel](key)).
		ChangeErr(func(_ gorp.Context, c Channel) (Channel, error) {
			if !c.IsCalculated() {
				return Channel{}, errors.Wrapf(
					validate.ErrValidation,
					"cannot change the data type of non-calculated channel %q",
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
// also deletes the index of each deleted calculated channel that no other channel uses.
// Unless allowInternal is true, deleting any internal channel returns an error and no
// channels are deleted.
func (w Writer) DeleteMany(ctx context.Context, keys []Key, allowInternal bool) error {
	keys, err := w.withCalculatedIndexes(ctx, keys)
	if err != nil {
		return err
	}
	return w.delete(ctx, keys, allowInternal)
}

// DeleteManyByNames deletes the channels with the given names. Unless allowInternal is
// true, deleting any internal channel returns an error and no channels are deleted.
func (w Writer) DeleteManyByNames(
	ctx context.Context, names []string, allowInternal bool,
) error {
	var res []Channel
	if err := w.svc.newRetrieve().
		Where(MatchNames(names...)).
		Entries(&res).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.DeleteMany(ctx, KeysFromChannels(res), allowInternal)
}

// withCalculatedIndexes returns keys and the index of each calculated channel in keys
// that no channel outside keys uses.
func (w Writer) withCalculatedIndexes(ctx context.Context, keys Keys) (Keys, error) {
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
		if c.IsCalculated() && c.LocalIndex != 0 && !deleting.Contains(c.Index()) {
			indexes.Add(c.Index())
		}
	}
	used, err := w.usedIndexes(ctx, indexes, deleting)
	if err != nil {
		return nil, err
	}
	return slices.Concat(keys, indexes.Difference(used).Slice()), nil
}

// usedIndexes returns the keys in indexes that a channel outside exclude uses as its
// index. The Core keeps such an index where it is when it moves or deletes the
// calculated channel that owns it.
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
	return w.RenameMany(ctx, []Key{key}, []string{newName}, allowInternal)
}

// RenameMany renames the channels with the given keys to the corresponding entries in
// newNames, which must be parallel to keys, and attaches a rename action for each one.
// Unless allowInternal is true, renaming any internal channel returns an error.
func (w Writer) RenameMany(
	ctx context.Context, keys []Key, newNames []string, allowInternal bool,
) error {
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

// create orchestrates channel creation: it validates names, preprocesses calculated
// channels, resolves existing channels, allocates keys through the distribution
// allocator, and writes the rich channel records to the table.
func (w Writer) create(
	ctx context.Context,
	_channels *[]Channel,
	opts createOptions,
) error {
	channels := *_channels
	if err := w.validateCreate(ctx, channels, opts); err != nil {
		return err
	}
	if err := w.prepareCreate(channels); err != nil {
		return err
	}
	if err := w.resetStoredKeys(ctx, channels, opts); err != nil {
		return err
	}
	// Reset channels that already exist by name to their stored records before index
	// synthesis, so an existing calculated channel keeps its stored index.
	if opts.retrieveIfNameExists {
		if err := w.resetStoredNames(ctx, channels); err != nil {
			return err
		}
	}
	// Auto-create index channels for calculated channels that do not yet have one (both
	// brand-new calculated channels and existing ones missing an index). Synthesis
	// runs before the overwrite step so the auto-created index names flow through the
	// same name-collision machinery as caller-provided channels.
	indexChannels, calcNeedingIndex, err := synthesizeIndexes(channels)
	if err != nil {
		return err
	}
	regroup, err := w.adoptIndexes(ctx, indexChannels, opts)
	if err != nil {
		return err
	}
	channels = append(channels, indexChannels...)
	if opts.overwriteIfNameExistsAndDifferentProperties {
		if err = w.overwriteDiffering(ctx, channels); err != nil {
			return err
		}
	}
	written, err := w.allocateNew(ctx, channels)
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
	return w.attachCreates(ctx, lo.Uniq(written))
}

// validateCreate validates each channel, its group, and the names of channels.
func (w Writer) validateCreate(
	ctx context.Context,
	channels []Channel,
	opts createOptions,
) error {
	for i, ch := range channels {
		if err := ch.Validate(); err != nil {
			return validate.PathedError(err, strconv.Itoa(i))
		}
		if err := group.CheckMember(
			ctx,
			w.tx,
			ch.Group,
			resource.TypeChannel,
			nil,
		); err != nil {
			return validate.PathedError(err, strconv.Itoa(i))
		}
	}
	if !*w.svc.cfg.ValidateNames {
		return nil
	}
	skipExisting := opts.retrieveIfNameExists ||
		opts.overwriteIfNameExistsAndDifferentProperties
	return w.validateChannelNames(
		ctx,
		KeysFromChannels(channels),
		Names(channels),
		skipExisting,
	)
}

// prepareCreate sets the leaseholder of each channel. A calculated channel is virtual
// and free, and only a calculated channel keeps its local key.
func (w Writer) prepareCreate(channels []Channel) error {
	for i, ch := range channels {
		if ch.Leaseholder == 0 {
			channels[i].Leaseholder = w.svc.cfg.HostKey
		}
		if ch.IsCalculated() {
			if ch.LocalIndex != 0 && ch.LocalKey == 0 {
				return validate.PathedError(
					errors.Wrap(
						validate.ErrValidation,
						"calculated channels cannot specify an index manually",
					),
					"local_index",
				)
			}
			channels[i].Leaseholder = node.KeyFree
			channels[i].Virtual = true
		} else if ch.LocalKey != 0 {
			channels[i].LocalKey = 0
		}
	}
	return nil
}

// resetStoredKeys resets each channel whose key exists to its stored record. Only a
// calculated channel keeps a key at this point. A stored key is an error unless the
// create retrieves channels that exist.
func (w Writer) resetStoredKeys(
	ctx context.Context,
	channels []Channel,
	opts createOptions,
) error {
	keys := KeysFromChannels(channels)
	keyed := lo.Filter(keys, func(k Key, _ int) bool { return k.LocalKey() != 0 })
	if len(keyed) == 0 {
		return nil
	}
	var stored []Channel
	if err := w.svc.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Channel](keyed...)).
		SkipMissingKeys().
		Entries(&stored).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	keyPositions := positions(keys)
	for _, c := range stored {
		if !opts.retrieveIfNameExists {
			return errors.Wrapf(
				validate.ErrValidation,
				"%s %s already exists",
				tableName,
				c.Key(),
			)
		}
		channels[keyPositions[c.Key()][0]] = c
	}
	return nil
}

// resetStoredNames resets each channel whose name exists to its stored record.
func (w Writer) resetStoredNames(ctx context.Context, channels []Channel) error {
	names := Names(channels)
	var existing []Channel
	if err := w.svc.newRetrieve().
		Where(MatchNames(names...)).
		Entries(&existing).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	namePositions := positions(names)
	for _, e := range existing {
		channels[namePositions[e.Name][0]] = e
	}
	return nil
}

// synthesizeIndexes returns a new index channel for each calculated channel without
// an index, and the positions in channels of the calculated channels that need one.
func synthesizeIndexes(channels []Channel) ([]Channel, []int, error) {
	indexChannels := make([]Channel, 0, len(channels))
	calcNeedingIndex := make([]int, 0)
	batchPositionByName := make(map[string]int, len(channels))
	for i, ch := range channels {
		batchPositionByName[ch.Name] = i
	}
	for i, ch := range channels {
		if !ch.IsCalculated() || ch.LocalIndex != 0 {
			continue
		}
		indexName := ch.Name + calculatedIndexNameSuffix
		if bi, inBatch := batchPositionByName[indexName]; inBatch {
			// An in-batch channel can only serve as the index if it satisfies the same
			// predicate as an adopted existing index. Adopting a host-leased or
			// persisted index would graft its LocalKey into the calculated channel's
			// free keyspace, producing an index key that resolves to the wrong channel.
			if !isFreeIndex(channels[bi]) {
				return nil, nil, errors.Wrapf(
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
			Internal:    ch.Internal,
			Group:       ch.Group,
		})
		calcNeedingIndex = append(calcNeedingIndex, i)
	}
	return indexChannels, calcNeedingIndex, nil
}

// isFreeIndex reports whether ch can serve as the index of a calculated channel.
func isFreeIndex(ch Channel) bool {
	return ch.IsIndex && ch.Virtual && ch.Leaseholder == node.KeyFree &&
		ch.DataType == telem.TimestampT
}

// adoptIndexes resolves the names of indexChannels against existing channels. A
// compatible index, such as one left behind by a deleted calculated channel, replaces
// its entry in indexChannels instead of a duplicate name. An incompatible channel is a
// validation error, unless the create overwrites it. adoptIndexes returns the position
// in indexChannels of each adopted index to move to the group of its new calculated
// channel.
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
		if opts.overwriteIfNameExistsAndDifferentProperties {
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

// overwriteDiffering deletes existing channels of the same name whose properties
// differ from the ones being created, so the create can replace them. Channels that
// match an existing record (ignoring allocated keys) are reset to that record instead.
func (w Writer) overwriteDiffering(ctx context.Context, channels []Channel) error {
	names := Names(channels)
	if len(names) == 0 {
		return nil
	}
	var existing []Channel
	if err := w.svc.newRetrieve().
		Where(MatchNames(names...)).
		Entries(&existing).
		Exec(ctx, w.tx); err != nil {
		return errors.Skip(err, query.ErrNotFound)
	}
	keysToDelete := make(Keys, 0, len(existing))
	namePositions := positions(names)
	for _, ex := range existing {
		i, found := lo.Find(namePositions[ex.Name], func(i int) bool {
			return channels[i].Key() != ex.Key()
		})
		if !found {
			continue
		}
		if channels[i].Equals(ex, "LocalKey", "LocalIndex", "Leaseholder") {
			channels[i] = ex
			continue
		}
		keysToDelete = append(keysToDelete, ex.Key())
	}
	if len(keysToDelete) == 0 {
		return nil
	}
	return w.delete(ctx, keysToDelete, true)
}

// allocateNew allocates local keys for the channels without one through the
// distribution allocator, writes them, and returns their keys.
func (w Writer) allocateNew(ctx context.Context, channels []Channel) (Keys, error) {
	newIndices := make([]int, 0, len(channels))
	minimal := make([]channel.Channel, 0, len(channels))
	for i, ch := range channels {
		if ch.LocalKey == 0 {
			newIndices = append(newIndices, i)
			minimal = append(minimal, ch.Distribution())
		}
	}
	if err := w.allocateAndWrite(ctx, channels, newIndices, minimal); err != nil {
		return nil, err
	}
	written := make(Keys, 0, len(newIndices))
	for _, i := range newIndices {
		written = append(written, channels[i].Key())
	}
	return written, nil
}

// linkStoredIndexes persists the index of each stored calculated channel at the
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
		w.dispatcher.Attach(w.tx, c.Key(), NewCreateAction(CreatePayload{Channel: c}))
	}
	return nil
}

// allocateAndWrite enforces the channel limit, then allocates keys for the new channels
// and writes them to the table. When the transaction commits, it creates their storage
// channels and records them in the external non-virtual set.
func (w Writer) allocateAndWrite(
	ctx context.Context,
	channels []Channel,
	newIndices []int,
	minimal []channel.Channel,
) error {
	if err := w.checkLimit(channels, newIndices); err != nil {
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

	linkCalculatedIndexes(channels, toCreate)

	externalCreatedKeys := make(Keys, 0, len(toCreate))
	for _, ch := range toCreate {
		if !ch.Internal && !ch.Virtual {
			externalCreatedKeys = append(externalCreatedKeys, ch.Key())
		}
	}
	// A data channel's index must exist before the data channel is written.
	slices.SortStableFunc(toCreate, indexesFirst)
	if err := w.svc.table.NewCreate().
		Exclusive().
		Entries(&toCreate).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	gorp.BeforeCommit(w.tx, func(ctx context.Context) error {
		if err := w.svc.cfg.Channel.Create(ctx, allocated); err != nil {
			return err
		}
		w.svc.mu.Lock()
		w.svc.mu.externalNonVirtualSet.Insert(externalCreatedKeys...)
		w.svc.mu.Unlock()
		return nil
	})
	return nil
}

// checkLimit checks that the external, non-virtual channels at newIndices in channels
// fit the channel limit.
func (w Writer) checkLimit(channels []Channel, newIndices []int) error {
	externalNewCount := 0
	for _, i := range newIndices {
		if !channels[i].Internal && !channels[i].Virtual {
			externalNewCount++
		}
	}
	w.svc.mu.RLock()
	count := w.svc.mu.externalNonVirtualSet.Size()
	w.svc.mu.RUnlock()
	return w.svc.cfg.Limit(types.Uint20(int(count) + externalNewCount))
}

// linkCalculatedIndexes links each calculated channel without an index in channels and
// toCreate to its index channel in channels, by setting the LocalIndex of the
// calculated channel to the LocalKey of the index.
func linkCalculatedIndexes(channels, toCreate []Channel) {
	indexKeyByName := make(map[string]LocalKey, len(channels))
	for _, ch := range channels {
		if ch.IsIndex {
			indexKeyByName[ch.Name] = ch.LocalKey
		}
	}
	link := func(chs []Channel) {
		for i := range chs {
			if !chs[i].IsCalculated() || chs[i].LocalIndex != 0 {
				continue
			}
			if k, ok := indexKeyByName[chs[i].Name+calculatedIndexNameSuffix]; ok {
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

// validateChannelNames rejects a create/rename request whose proposed names are not
// identifiers, duplicate each other, or collide with an existing channel under a
// different key. It leaves an empty name to Channel.Validate.
func (w Writer) validateChannelNames(
	ctx context.Context,
	keys Keys,
	names []string,
	skipExisting bool,
) error {
	if err := validateNameSyntax(names); err != nil {
		return err
	}
	if skipExisting {
		return nil
	}
	var conflictingChannels []Channel
	if err := w.svc.newRetrieve().
		Where(MatchNames(names...)).
		Entries(&conflictingChannels).
		Exec(ctx, w.tx); err != nil {
		return errors.Skip(err, query.ErrNotFound)
	}
	nameConflicts := make(map[string]int, len(conflictingChannels))
	for i, ch := range conflictingChannels {
		nameConflicts[ch.Name] = i
	}
	for i, name := range names {
		conflictingIdx, conflict := nameConflicts[name]
		if !conflict {
			continue
		}
		existingCh := conflictingChannels[conflictingIdx]
		if existingCh.Key() == keys[i] {
			continue
		}
		return validate.PathedError(
			errors.Wrapf(
				validate.ErrValidation, "channel with name '%s' already exists", name,
			),
			fmt.Sprintf("[%d].name", i),
		)
	}
	return nil
}

// validateNameSyntax rejects names that are not identifiers or that repeat. It
// leaves an empty name to Channel.Validate.
func validateNameSyntax(names []string) error {
	for i, name := range names {
		if name != "" && !validate.IsIdentifier(name) {
			return validate.PathedError(
				errors.Wrapf(
					validate.ErrValidation,
					"channel name '%s' contains invalid characters. Only letters, digits, and underscores are allowed, and it cannot start with a digit",
					name,
				),
				fmt.Sprintf("[%d]", i), "name",
			)
		}
	}
	namesSeen := make(set.Set[string], len(names))
	for i, name := range names {
		if namesSeen.Contains(name) {
			return validate.PathedError(
				errors.Wrapf(
					validate.ErrValidation,
					"duplicate channel name '%s' in request",
					name,
				),
				fmt.Sprintf("[%d].name", i),
			)
		}
		namesSeen.Add(name)
	}
	return nil
}

func (w Writer) delete(ctx context.Context, keys Keys, allowInternal bool) error {
	if err := w.svc.table.NewDelete().
		Where(gorp.MatchKeys[Key, Channel](keys...)).
		Guard(func(_ gorp.Context, c Channel) error {
			if c.Internal && !allowInternal {
				return errors.Wrapf(
					validate.ErrValidation,
					"can't delete internal channel %q",
					c.Name,
				)
			}
			return nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	gorp.BeforeCommit(w.tx, func(ctx context.Context) error {
		if err := w.svc.cfg.Channel.Delete(ctx, keys); err != nil {
			return err
		}
		w.svc.mu.Lock()
		w.svc.mu.externalNonVirtualSet.Remove(keys...)
		w.svc.mu.Unlock()
		return nil
	})
	return nil
}

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
	if *w.svc.cfg.ValidateNames {
		if err := w.validateChannelNames(ctx, keys, names, false); err != nil {
			return err
		}
	}
	renameMap := make(map[Key]string, len(keys))
	for i, key := range keys {
		renameMap[key] = names[i]
	}
	if err := w.svc.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Channel](keys...)).
		ChangeErr(func(_ gorp.Context, c Channel) (Channel, error) {
			if c.Internal && !allowInternal {
				return Channel{},
					errors.Wrapf(
						validate.ErrValidation,
						"cannot rename internal channel %v",
						c,
					)
			}
			c.Name = renameMap[c.Key()]
			if err := c.Validate(); err != nil {
				return Channel{}, err
			}
			return c, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	for i, key := range keys {
		w.svc.renames.Add(w.tx, rename{key: key, name: names[i]})
	}
	if len(keys) > 0 {
		count := len(w.svc.renames.Pending(w.tx))
		gorp.BeforeCommit(w.tx, func(ctx context.Context) error {
			return w.renameStored(ctx, count)
		})
	}
	return nil
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
	return w.svc.cfg.Channel.Rename(ctx, names)
}

// positions maps each value in vals to the positions that hold it, in ascending order.
func positions[T comparable](vals []T) map[T][]int {
	pos := make(map[T][]int, len(vals))
	for i, v := range vals {
		pos[v] = append(pos[v], i)
	}
	return pos
}
