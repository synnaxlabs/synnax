// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, updates, and deletes ICDs within a transaction.
type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	table     *gorp.Table[Key, ICD]
}

// Create creates the ICD, or replaces it when an ICD with its key exists. A replacement
// must carry the Hash of the ICD as the writer read it, or Create returns an error
// wrapping ErrStale. Create assigns keys to the ICD, its messages, and their fields
// where absent, and sets Hash to the hash of the stored ICD. It returns a path-scoped
// validation error for each problem Check finds.
func (w Writer) Create(ctx context.Context, ic *ICD) error {
	var prev *ICD
	if ic.Key != uuid.Nil() {
		existing, err := w.retrieve(ctx, ic.Key)
		if err == nil {
			prev = &existing
		} else if !errors.Is(err, query.ErrNotFound) {
			return err
		}
	}
	if prev != nil {
		if ic.Hash != prev.Hash {
			return errors.Wrapf(ErrStale, "ICD %s", prev.Name)
		}
	} else if ic.Key == uuid.Nil() {
		ic.Key = uuid.New()
	}
	assignKeys(ic)
	if err := Check(*ic).Err(); err != nil {
		return err
	}
	var err error
	if ic.Hash, err = Hash(*ic); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(ic).Exec(ctx, w.tx); err != nil {
		return err
	}
	if prev != nil {
		return nil
	}
	return w.otgWriter.DefineResources(ctx, ic.OntologyID())
}

// CreateMany creates or replaces each of the given ICDs as Create does.
func (w Writer) CreateMany(ctx context.Context, ics *[]ICD) error {
	for i := range *ics {
		if err := w.Create(ctx, &(*ics)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Import replaces the messages of the ICD with the given key by those parsed from data
// in the given format. Messages and fields keep their keys when their names match (see
// Merge). Import returns the updated ICD, a path-scoped validation error when data or
// the result is invalid, and an error wrapping query.ErrNotFound when no ICD has the
// key.
func (w Writer) Import(
	ctx context.Context,
	key Key,
	format Format,
	data []byte,
) (ICD, error) {
	messages, err := Parse(format, data)
	if err != nil {
		return ICD{}, err
	}
	existing, err := w.retrieve(ctx, key)
	if err != nil {
		return ICD{}, err
	}
	ic := Merge(existing, messages)
	if err = w.Create(ctx, &ic); err != nil {
		return ICD{}, err
	}
	return ic, nil
}

// Rename changes the name of the ICD with the given key. It returns an error wrapping
// query.ErrNotFound when no ICD has the key.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	if name == "" {
		return validate.PathedError(
			errors.Wrap(validate.ErrValidation, "name is required"),
			"name",
		)
	}
	ic, err := w.retrieve(ctx, key)
	if err != nil {
		return err
	}
	ic.Name = name
	return w.table.NewCreate().Entry(&ic).Exec(ctx, w.tx)
}

// Delete deletes the ICDs with the given keys. Delete is idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, ICD](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...)
}

func (w Writer) retrieve(ctx context.Context, key Key) (ICD, error) {
	var ic ICD
	err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, ICD](key)).
		Entry(&ic).
		Exec(ctx, w.tx)
	return ic, err
}

// assignKeys gives a new key to every message and field that has none. It skips text
// fields with no encoding, which Check rejects.
func assignKeys(ic *ICD) {
	for i := range ic.Messages {
		m := &ic.Messages[i]
		if m.Key == uuid.Nil() {
			m.Key = uuid.New()
		}
		m.Payload.UpdateFieldBases(func(fb *BaseField) {
			if fb.Key == uuid.Nil() {
				fb.Key = uuid.New()
			}
		})
	}
}
