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
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Resolver completes bus task configs from the ICD they reference. Config stores call
// it before storing a config.
type Resolver struct {
	// Stamper stamps the hash of the ICD entries the config uses into the config and
	// relates the task to the ICD.
	Stamper icd.Stamper
}

// Override returns r with each nil dependency taken from other.
func (r Resolver) Override(other Resolver) Resolver {
	r.Stamper.DB = override.Nil(r.Stamper.DB, other.Stamper.DB)
	r.Stamper.Ontology = override.Nil(r.Stamper.Ontology, other.Stamper.Ontology)
	return r
}

// Validate returns a validation error if r is missing a dependency.
func (r Resolver) Validate() error {
	v := validate.New("bus.resolver")
	v.NotNil("stamper.db", r.Stamper.DB)
	v.NotNil("stamper.ontology", r.Stamper.Ontology)
	return v.Error()
}

// Read stamps cfg and returns a path-scoped validation error when cfg has messages but
// no ICD, or when a message or field it reads is not in its ICD.
func (r Resolver) Read(
	ctx context.Context,
	tx gorp.Tx,
	taskKey uuid.UUID,
	cfg *ReadConfig,
) error {
	keys := lo.Map(cfg.Messages, func(m ReadMessage, _ int) icd.EntryKey {
		return m.Message
	})
	messages, err := r.stamp(ctx, tx, taskKey, &cfg.Reference, keys)
	if err != nil {
		return err
	}
	for i, m := range cfg.Messages {
		fields := lo.Map(m.Fields, func(f ReadField, _ int) icd.FieldKey {
			return f.Field
		})
		if err = checkMessage(messages, i, m.Message, fields); err != nil {
			return err
		}
	}
	return nil
}

// Write stamps cfg and returns a path-scoped validation error when cfg has messages but
// no ICD, or when a message or field it sends is not in its ICD.
func (r Resolver) Write(
	ctx context.Context,
	tx gorp.Tx,
	taskKey uuid.UUID,
	cfg *WriteConfig,
) error {
	keys := lo.Map(cfg.Messages, func(m WriteMessage, _ int) icd.EntryKey {
		return m.Message
	})
	messages, err := r.stamp(ctx, tx, taskKey, &cfg.Reference, keys)
	if err != nil {
		return err
	}
	for i, m := range cfg.Messages {
		fields := lo.Map(m.Fields, func(f WriteField, _ int) icd.FieldKey {
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
	ref *icd.Reference,
	keys []icd.EntryKey,
) (map[icd.EntryKey]icd.MessageEntry, error) {
	if ref.ICD == nil && len(keys) > 0 {
		return nil, validate.PathedError(errors.Wrap(
			validate.ErrValidation, "select an ICD before choosing messages",
		), "icd")
	}
	ic, err := r.Stamper.Stamp(ctx, tx, taskKey, ref, keys)
	if err != nil {
		return nil, err
	}
	messages := make(map[icd.EntryKey]icd.MessageEntry)
	for _, e := range ic.Entries {
		if m, ok := e.Variant.(icd.MessageEntry); ok {
			messages[m.Key] = m
		}
	}
	return messages, nil
}

func checkMessage(
	messages map[icd.EntryKey]icd.MessageEntry,
	index int,
	key icd.EntryKey,
	fields []icd.FieldKey,
) error {
	path := "messages." + strconv.Itoa(index)
	m, ok := messages[key]
	if !ok {
		return validate.PathedError(errors.Wrapf(
			validate.ErrValidation, "message %s is not in the ICD", key,
		), path+".message")
	}
	bases := m.Payload.FieldBases()
	known := make(set.Set[icd.FieldKey], len(bases))
	for _, b := range bases {
		known.Add(b.Key)
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
