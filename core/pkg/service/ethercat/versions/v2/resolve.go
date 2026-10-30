// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/x/errors"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

// Entry is one PDO entry that a slave's scan discovered.
type Entry struct {
	// Name is the entry's name, which the slave may repeat across PDOs.
	Name string `json:"name"`
	// Index is the object dictionary index of the entry.
	Index uint16 `json:"index"`
	// SubIndex is the object dictionary subindex of the entry.
	SubIndex uint8 `json:"sub_index"`
	// DataType is the data type of the entry.
	DataType telem.DataType `json:"data_type"`
}

// Slave lists the PDO entries of one slave.
type Slave struct {
	// Inputs are the entries the slave sends to the master.
	Inputs []Entry `json:"inputs"`
	// Outputs are the entries the master sends to the slave.
	Outputs []Entry `json:"outputs"`
}

// Slaves returns the slaves among keys, read through tx. A key with no slave is absent
// from the result.
type Slaves func(
	ctx context.Context,
	tx gorp.Tx,
	keys []string,
) (map[string]Slave, error)

// ResolveReadConfig addresses each unresolved channel in cfg to the input entry its
// slave lists under the channel's entry name. A channel stays unresolved when the name
// matches no entry, or entries at more than one address.
func ResolveReadConfig(
	ctx context.Context,
	tx gorp.Tx,
	slaves Slaves,
	cfg *ReadConfig,
) error {
	found, err := ReadSlaves(
		ctx,
		tx,
		slaves,
		lo.FilterMap(cfg.Channels, func(ch ReadChannel, _ int) (string, bool) {
			u, ok := ch.Variant.(UnresolvedReadChannel)
			return u.Device, ok
		}),
	)
	if err != nil {
		return err
	}
	for i, ch := range cfg.Channels {
		u, ok := ch.Variant.(UnresolvedReadChannel)
		if !ok {
			continue
		}
		if e, ok := FindEntry(found[u.Device].Inputs, u.Pdo); ok {
			cfg.Channels[i].Variant = EntryReadChannel{
				BaseReadChannel: u.BaseReadChannel,
				Index:           e.Index,
				SubIndex:        e.SubIndex,
				DataType:        e.DataType,
			}
		}
	}
	return nil
}

// ResolveWriteConfig addresses each unresolved channel in cfg to the output entry its
// slave lists under the channel's entry name. A channel stays unresolved when the name
// matches no entry, or entries at more than one address.
func ResolveWriteConfig(
	ctx context.Context,
	tx gorp.Tx,
	slaves Slaves,
	cfg *WriteConfig,
) error {
	found, err := ReadSlaves(
		ctx,
		tx,
		slaves,
		lo.FilterMap(cfg.Channels, func(ch WriteChannel, _ int) (string, bool) {
			u, ok := ch.Variant.(UnresolvedWriteChannel)
			return u.Device, ok
		}),
	)
	if err != nil {
		return err
	}
	for i, ch := range cfg.Channels {
		u, ok := ch.Variant.(UnresolvedWriteChannel)
		if !ok {
			continue
		}
		if e, ok := FindEntry(found[u.Device].Outputs, u.Pdo); ok {
			cfg.Channels[i].Variant = EntryWriteChannel{
				BaseWriteChannel: u.BaseWriteChannel,
				Index:            e.Index,
				SubIndex:         e.SubIndex,
				DataType:         e.DataType,
			}
		}
	}
	return nil
}

// NewReadMigration returns the migration that lifts stored read configs to v2 and
// resolves their channels against slaves.
func NewReadMigration(slaves Slaves) gorp.Migration {
	return newMigration(
		"v60_ethercat_read_entry_address",
		slaves,
		MigrateReadConfig,
		ResolveReadConfig,
	)
}

// NewWriteMigration returns the migration that lifts stored write configs to v2 and
// resolves their channels against slaves.
func NewWriteMigration(slaves Slaves) gorp.Migration {
	return newMigration(
		"v60_ethercat_write_entry_address",
		slaves,
		MigrateWriteConfig,
		ResolveWriteConfig,
	)
}

func newMigration[I, O gorp.Entry[uuid.UUID]](
	key string,
	slaves Slaves,
	lift func(context.Context, I) (O, error),
	resolve func(context.Context, gorp.Tx, Slaves, *O) error,
) gorp.Migration {
	return migrate.New(
		key,
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			var olds []I
			if err := gorp.NewRetrieve[uuid.UUID, I]().
				Entries(&olds).
				Exec(ctx, tx); err != nil {
				return err
			}
			news := make([]O, 0, len(olds))
			for _, old := range olds {
				cfg, err := lift(ctx, old)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				if err = resolve(ctx, tx, slaves, &cfg); err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				news = append(news, cfg)
			}
			return gorp.WrapWriter[uuid.UUID, O](tx).Set(ctx, news...)
		},
	)
}

// FindEntry returns the entry in entries named name. ok is false when the name matches
// no entry, or entries at more than one address.
func FindEntry(entries []Entry, name string) (Entry, bool) {
	matches := lo.UniqBy(
		lo.Filter(entries, func(e Entry, _ int) bool { return e.Name == name }),
		func(e Entry) [2]int { return [2]int{int(e.Index), int(e.SubIndex)} },
	)
	if len(matches) != 1 {
		return Entry{}, false
	}
	return matches[0], true
}

// ReadSlaves returns the slaves among keys, read through tx. It reads nothing when keys
// is empty.
func ReadSlaves(
	ctx context.Context,
	tx gorp.Tx,
	slaves Slaves,
	keys []string,
) (map[string]Slave, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	return slaves(ctx, tx, lo.Uniq(keys))
}
