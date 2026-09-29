// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bus

import (
	"context"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Resolver completes bus task configs from the library they reference. Config stores
// call it before storing a config.
type Resolver struct {
	// Stamper stamps the library hash into the config and relates the task to the
	// library.
	Stamper library.Stamper
}

// Read stamps cfg and returns a path-scoped validation error when a message or field
// it reads is not in its library.
func (r Resolver) Read(
	ctx context.Context,
	tx gorp.Tx,
	taskKey uuid.UUID,
	cfg *ReadConfig,
) error {
	messages, err := r.stamp(ctx, tx, taskKey, &cfg.Reference)
	if err != nil {
		return err
	}
	for i, m := range cfg.Messages {
		fields := lo.Map(m.Fields, func(f ReadField, _ int) library.FieldKey {
			return f.Field
		})
		if err = checkMessage(messages, i, m.Message, fields); err != nil {
			return err
		}
	}
	return nil
}

// Write stamps cfg and returns a path-scoped validation error when a message or field
// it sends is not in its library.
func (r Resolver) Write(
	ctx context.Context,
	tx gorp.Tx,
	taskKey uuid.UUID,
	cfg *WriteConfig,
) error {
	messages, err := r.stamp(ctx, tx, taskKey, &cfg.Reference)
	if err != nil {
		return err
	}
	for i, m := range cfg.Messages {
		fields := lo.Map(m.Fields, func(f WriteField, _ int) library.FieldKey {
			return f.Field
		})
		if err = checkMessage(messages, i, m.Message, fields); err != nil {
			return err
		}
	}
	return nil
}

func (r Resolver) stamp(
	ctx context.Context,
	tx gorp.Tx,
	taskKey uuid.UUID,
	ref *library.Reference,
) (map[library.EntryKey]library.MessageEntry, error) {
	l, err := r.Stamper.Stamp(ctx, tx, taskKey, ref)
	if err != nil {
		return nil, err
	}
	messages := make(map[library.EntryKey]library.MessageEntry)
	for _, e := range l.Entries {
		if m, ok := e.Variant.(library.MessageEntry); ok {
			messages[m.Key] = m
		}
	}
	return messages, nil
}

func checkMessage(
	messages map[library.EntryKey]library.MessageEntry,
	index int,
	key library.EntryKey,
	fields []library.FieldKey,
) error {
	path := "messages." + strconv.Itoa(index)
	m, ok := messages[key]
	if !ok {
		return validate.PathedError(errors.Wrapf(
			validate.ErrValidation, "message %s is not in the library", key,
		), path+".message")
	}
	known := make(set.Set[library.FieldKey], len(m.Fields))
	for _, f := range m.Fields {
		if base, ok := library.FieldBase(f); ok {
			known.Add(base.Key)
		}
	}
	for j, f := range fields {
		if !known.Contains(f) {
			return validate.PathedError(errors.Wrapf(
				validate.ErrValidation, "field %s is not in message %s", f, m.Name,
			), path+".fields."+strconv.Itoa(j)+".field")
		}
	}
	return nil
}
