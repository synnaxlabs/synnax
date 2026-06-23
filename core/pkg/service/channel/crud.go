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

	"github.com/samber/lo"
	dischannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	xtypes "github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

const calculatedIndexNameSuffix = "_time"

// create orchestrates channel creation: it validates names, preprocesses calculated
// channels, resolves existing channels, allocates keys and storage through the
// distribution allocator, writes the rich channel records to the table, and registers
// ontology resources.
func (s *Service) create(ctx context.Context, tx gorp.Tx, _channels *[]Channel, opts CreateOptions) error {
	channels := *_channels
	if *s.cfg.ValidateNames {
		skipExisting := opts.retrieveIfNameExists || opts.overwriteIfNameExistsAndDifferentProperties
		if err := s.validateChannelNames(ctx, tx, KeysFromChannels(channels), Names(channels), skipExisting); err != nil {
			return err
		}
	}
	for i, ch := range channels {
		if ch.Leaseholder == 0 {
			channels[i].Leaseholder = s.cfg.HostResolver.HostKey()
		}
		if ch.IsCalculated() {
			if ch.LocalIndex != 0 && ch.LocalKey == 0 {
				return validate.PathedError(
					errors.Wrap(validate.ErrValidation, "calculated channels cannot specify an index manually"),
					"local_index",
				)
			}
			channels[i].Leaseholder = node.KeyFree
			channels[i].Virtual = true
		} else if ch.LocalKey != 0 {
			channels[i].LocalKey = 0
		}
	}

	// Update existing channels passed in with a key (e.g. a calculated channel whose
	// expression is being changed). Reset to the stored record when RetrieveIfNameExists
	// so the caller does not mistake an update for a no-op create.
	if err := s.updateExisting(ctx, tx, &channels, opts); err != nil {
		return err
	}

	if opts.overwriteIfNameExistsAndDifferentProperties {
		if err := s.deleteOverwritten(ctx, tx, &channels); err != nil {
			return err
		}
	}

	// Auto-create index channels for calculated channels that do not yet have one (both
	// brand-new calculated channels and existing ones missing an index).
	indexChannels := make([]Channel, 0, len(channels))
	calcNeedingIndex := make([]int, 0)
	for i, ch := range channels {
		if ch.IsCalculated() && ch.LocalIndex == 0 {
			indexChannels = append(indexChannels, Channel{
				Name:        ch.Name + calculatedIndexNameSuffix,
				DataType:    telem.TimeStampT,
				IsIndex:     true,
				Virtual:     true,
				Leaseholder: node.KeyFree,
				Internal:    ch.Internal,
			})
			calcNeedingIndex = append(calcNeedingIndex, i)
		}
	}
	channels = append(channels, indexChannels...)

	if err := s.validateFreeVirtual(channels); err != nil {
		return err
	}

	toCreate, err := s.resolveExistingAndAssignKeys(ctx, tx, &channels, opts.retrieveIfNameExists)
	if err != nil {
		return err
	}

	s.linkCalculatedIndexes(toCreate, channels)

	if err := s.writeChannels(ctx, tx, toCreate); err != nil {
		return err
	}

	// Persist updated LocalIndex links for existing calculated channels whose index was
	// just created.
	if err := s.updateCalculatedIndexLinks(ctx, tx, channels, calcNeedingIndex); err != nil {
		return err
	}

	*_channels = channels
	return s.maybeSetResources(ctx, tx, channels, opts)
}

// updateExisting updates the stored records of channels passed in with a non-zero key,
// applying name and (for calculated channels) expression/operations/index/data-type
// changes. When RetrieveIfNameExists is set the in-memory channel is reset to the stored
// record instead.
func (s *Service) updateExisting(ctx context.Context, tx gorp.Tx, channels *[]Channel, opts CreateOptions) error {
	keys := KeysFromChannels(*channels)
	existingKeys := lo.Filter(keys, func(k Key, _ int) bool { return k.LocalKey() != 0 })
	if len(existingKeys) == 0 {
		return nil
	}
	err := s.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Channel](existingKeys...)).
		ChangeErr(func(_ gorp.Context, c Channel) (Channel, error) {
			idx := lo.IndexOf(keys, c.Key())
			ic := (*channels)[idx]
			if opts.retrieveIfNameExists {
				(*channels)[idx] = c
				return c, nil
			}
			c.Name = ic.Name
			if c.IsCalculated() && ic.IsCalculated() {
				c.Expression = ic.Expression
				c.Operations = ic.Operations
				c.LocalIndex = ic.LocalIndex
				c.DataType = ic.DataType
			}
			return c, nil
		}).
		Exec(ctx, tx)
	if err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	return nil
}

// resolveExistingAndAssignKeys resolves channels that already exist by name (when
// retrieveIfNameExists is set, reusing their stored records) and allocates local keys
// plus storage for the remaining new channels through the distribution allocator. It
// returns the set of channels that were newly created.
func (s *Service) resolveExistingAndAssignKeys(
	ctx context.Context,
	tx gorp.Tx,
	channels *[]Channel,
	retrieveIfNameExists bool,
) (toCreate []Channel, err error) {
	if retrieveIfNameExists {
		names := Names(*channels)
		var existing []Channel
		if err = s.newRetrieve().
			Where(MatchNames(names...)).
			Entries(&existing).
			Exec(ctx, tx); err != nil {
			return nil, errors.Skip(err, query.ErrNotFound)
		}
		for _, e := range existing {
			idx := lo.IndexOf(names, e.Name)
			if idx < 0 {
				continue
			}
			(*channels)[idx] = e
		}
	}

	newIndices := make([]int, 0, len(*channels))
	minimal := make([]dischannel.Channel, 0, len(*channels))
	for i, ch := range *channels {
		if ch.LocalKey == 0 {
			newIndices = append(newIndices, i)
			minimal = append(minimal, ch.Distribution())
		}
	}
	if len(minimal) == 0 {
		return toCreate, nil
	}
	allocated, allocErr := s.cfg.Channel.Create(ctx, minimal)
	if allocErr != nil {
		return nil, allocErr
	}
	toCreate = make([]Channel, 0, len(newIndices))
	for j, i := range newIndices {
		(*channels)[i].LocalKey = allocated[j].LocalKey
		(*channels)[i].LocalIndex = allocated[j].LocalIndex
		toCreate = append(toCreate, (*channels)[i])
	}
	return toCreate, nil
}

// linkCalculatedIndexes links each calculated channel to its auto-created index channel
// by setting the calculated channel's LocalIndex to the index channel's assigned
// LocalKey. calcNeedingIndex holds the indices (into channels) of calculated channels
// that had an index auto-created for them.
func (s *Service) linkCalculatedIndexes(toCreate []Channel, channels []Channel) {
	indexKeyByName := make(map[string]LocalKey, len(channels))
	for _, ch := range channels {
		if ch.IsIndex {
			indexKeyByName[ch.Name] = ch.LocalKey
		}
	}
	link := func(set []Channel) {
		for i := range set {
			if !set[i].IsCalculated() || set[i].LocalIndex != 0 {
				continue
			}
			if k, ok := indexKeyByName[set[i].Name+calculatedIndexNameSuffix]; ok {
				set[i].LocalIndex = k
			}
		}
	}
	link(toCreate)
	link(channels)
}

// updateCalculatedIndexLinks persists the LocalIndex link for existing calculated
// channels (those already in the table) whose index channel was just created.
func (s *Service) updateCalculatedIndexLinks(ctx context.Context, tx gorp.Tx, channels []Channel, calcNeedingIndex []int) error {
	for _, idx := range calcNeedingIndex {
		ch := channels[idx]
		if ch.LocalKey == 0 || ch.LocalIndex == 0 {
			continue
		}
		exists, err := s.newRetrieve().Where(MatchKeys(ch.Key())).Exists(ctx, tx)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if err := s.table.NewUpdate().
			Where(gorp.MatchKeys[Key, Channel](ch.Key())).
			Change(func(_ gorp.Context, c Channel) Channel {
				c.LocalIndex = ch.LocalIndex
				return c
			}).
			Exec(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

// writeChannels enforces the external-channel overflow cap and writes the newly created
// channels to the table, updating the external/non-virtual key set.
func (s *Service) writeChannels(ctx context.Context, tx gorp.Tx, toCreate []Channel) error {
	externalCreatedKeys := make(Keys, 0, len(toCreate))
	for _, ch := range toCreate {
		if !ch.Internal && !ch.Virtual {
			externalCreatedKeys = append(externalCreatedKeys, ch.Key())
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	count := s.mu.externalNonVirtualSet.Size()
	if err := s.cfg.IntOverflowCheck(xtypes.Uint20(int(count) + len(externalCreatedKeys))); err != nil {
		return err
	}
	if err := s.table.NewCreate().Entries(&toCreate).Exec(ctx, tx); err != nil {
		return err
	}
	s.mu.externalNonVirtualSet.Insert(externalCreatedKeys...)
	return nil
}

// validateChannelNames rejects a create/rename request whose proposed names either
// duplicate each other or collide with an existing channel under a different key.
func (s *Service) validateChannelNames(
	ctx context.Context,
	tx gorp.Tx,
	keys Keys,
	names []string,
	skipExisting bool,
) error {
	for i, name := range names {
		if err := ValidateName(name); err != nil {
			return validate.PathedError(err, fmt.Sprintf("[%d].name", i))
		}
	}
	namesSeen := make(set.Set[string], len(names))
	for i, name := range names {
		if namesSeen.Contains(name) {
			return validate.PathedError(
				errors.Wrapf(validate.ErrValidation, "duplicate channel name '%s' in request", name),
				fmt.Sprintf("[%d].name", i),
			)
		}
		namesSeen.Add(name)
	}
	if skipExisting {
		return nil
	}
	var conflictingChannels []Channel
	if err := s.newRetrieve().
		Where(MatchNames(names...)).
		Entries(&conflictingChannels).
		Exec(ctx, tx); err != nil {
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
			errors.Wrapf(validate.ErrValidation, "channel with name '%s' already exists", name),
			fmt.Sprintf("[%d].name", i),
		)
	}
	return nil
}

func (s *Service) validateFreeVirtual(channels []Channel) error {
	for _, ch := range channels {
		if len(ch.Name) == 0 {
			return validate.PathedError(validate.ErrRequired, "name")
		}
	}
	return nil
}

func (s *Service) deleteOverwritten(ctx context.Context, tx gorp.Tx, channels *[]Channel) error {
	names := Names(*channels)
	if len(names) == 0 {
		return nil
	}
	var existing []Channel
	if err := s.newRetrieve().
		Where(MatchNames(names...)).
		Entries(&existing).
		Exec(ctx, tx); err != nil {
		return errors.Skip(err, query.ErrNotFound)
	}
	keysToDelete := make(Keys, 0, len(existing))
	for _, ex := range existing {
		ch, i, found := lo.FindIndexOf(*channels, func(ch Channel) bool {
			return ch.Name == ex.Name && ch.Key() != ex.Key()
		})
		if !found {
			continue
		}
		if ch.Equals(ex, "LocalKey", "LocalIndex", "Leaseholder") {
			(*channels)[i] = ex
			continue
		}
		keysToDelete = append(keysToDelete, ex.Key())
	}
	if len(keysToDelete) == 0 {
		return nil
	}
	if err := s.table.NewDelete().
		Where(gorp.MatchKeys[Key, Channel](keysToDelete...)).
		Exec(ctx, tx); err != nil {
		return err
	}
	return s.cfg.Channel.Delete(ctx, keysToDelete)
}

func (s *Service) maybeSetResources(
	ctx context.Context,
	txn gorp.Tx,
	channels []Channel,
	opts CreateOptions,
) error {
	if s.cfg.Ontology == nil || s.cfg.Group == nil {
		return nil
	}
	externalIDs := lo.FilterMap(channels, func(ch Channel, _ int) (ontology.ID, bool) {
		return OntologyID(ch.Key()), !ch.Internal
	})
	w := s.cfg.Ontology.NewWriter(txn)
	if err := w.DefineResource(ctx, externalIDs...); err != nil {
		return err
	}
	if opts.createWithoutGroupRelationship {
		return nil
	}
	return w.DefineRelationship(
		ctx,
		group.OntologyID(s.group.Key),
		ontology.RelationshipTypeParentOf,
		externalIDs...,
	)
}

func (s *Service) deleteByName(ctx context.Context, tx gorp.Tx, names []string, allowInternal bool) error {
	var res []Channel
	if err := s.newRetrieve().
		Where(MatchNames(names...)).
		Entries(&res).
		Exec(ctx, tx); err != nil {
		return errors.Skip(err, query.ErrNotFound)
	}
	return s.delete(ctx, tx, KeysFromChannels(res), allowInternal)
}

func (s *Service) delete(ctx context.Context, tx gorp.Tx, keys Keys, allowInternal bool) error {
	if !allowInternal {
		internalChannels := make([]Channel, 0, len(keys))
		if err := s.newRetrieve().
			Where(MatchKeys(keys...)).
			Where(MatchInternal(true)).
			Entries(&internalChannels).
			Exec(ctx, tx); err != nil {
			return errors.Skip(err, query.ErrNotFound)
		}
		if len(internalChannels) > 0 {
			names := make([]string, 0, len(internalChannels))
			for _, ch := range internalChannels {
				names = append(names, ch.Name)
			}
			return errors.Newf("can't delete internal channel(s): %v", names)
		}
	}
	if err := s.table.NewDelete().Where(gorp.MatchKeys[Key, Channel](keys...)).Exec(ctx, tx); err != nil {
		return err
	}
	if err := s.maybeDeleteResources(ctx, tx, keys); err != nil {
		return err
	}
	// Storage deletion goes last, as it is the only operation that can fail without an
	// atomic guarantee.
	if err := s.cfg.Channel.Delete(ctx, keys); err != nil {
		return err
	}
	s.mu.Lock()
	s.mu.externalNonVirtualSet.Remove(keys...)
	s.mu.Unlock()
	return nil
}

func (s *Service) maybeDeleteResources(ctx context.Context, tx gorp.Tx, keys Keys) error {
	if s.cfg.Ontology == nil {
		return nil
	}
	ids := lo.Map(keys, func(k Key, _ int) ontology.ID { return OntologyID(k) })
	w := s.cfg.Ontology.NewWriter(tx)
	return w.DeleteResource(ctx, ids...)
}

func channelNameUpdater(allowInternal bool, keys Keys, names []string) gorp.ChangeFunc[Key, Channel] {
	return func(_ gorp.Context, c Channel) (Channel, error) {
		if c.Internal && !allowInternal {
			return c, errors.Wrapf(validate.ErrValidation, "cannot rename internal channel %v", c)
		}
		c.Name = names[lo.IndexOf(keys, c.Key())]
		return c, nil
	}
}

func (s *Service) rename(
	ctx context.Context,
	tx gorp.Tx,
	keys Keys,
	names []string,
	allowInternal bool,
) error {
	if len(keys) != len(names) {
		return errors.Wrap(validate.ErrValidation, "keys and names must be the same length")
	}
	if *s.cfg.ValidateNames {
		if err := s.validateChannelNames(ctx, tx, keys, names, false); err != nil {
			return err
		}
	}
	if err := s.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Channel](keys...)).
		ChangeErr(channelNameUpdater(allowInternal, keys, names)).
		Exec(ctx, tx); err != nil {
		return err
	}
	return s.cfg.Channel.Rename(ctx, keys, names)
}
