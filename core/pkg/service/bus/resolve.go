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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
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
// no library, when a message or field it reads is not in its library, when fields
// that arrive at different times share an index, or when a polled reply with no
// identifier could be confused with a message the device sends unprompted.
func (r Resolver) Read(
	ctx context.Context,
	tx gorp.Tx,
	taskKey uuid.UUID,
	cfg *ReadConfig,
) error {
	messages, err := r.stamp(ctx, tx, taskKey, &cfg.Reference, len(cfg.Messages))
	if err != nil {
		return err
	}
	stamps := make(map[channel.Key]stamp)
	for i, m := range cfg.Messages {
		fields := lo.Map(m.Fields, func(f ReadField, _ int) library.FieldKey {
			return f.Field
		})
		if err = checkMessage(messages, i, m.Message, fields); err != nil {
			return err
		}
		if m.Disabled {
			continue
		}
		if err = checkIndexes(stamps, messages[m.Message], i, m.Fields); err != nil {
			return err
		}
	}
	return checkReplies(messages, cfg.Messages)
}

// Write stamps cfg and returns a path-scoped validation error when cfg has messages but
// no library, or when a message or field it sends is not in its library.
func (r Resolver) Write(
	ctx context.Context,
	tx gorp.Tx,
	taskKey uuid.UUID,
	cfg *WriteConfig,
) error {
	messages, err := r.stamp(ctx, tx, taskKey, &cfg.Reference, len(cfg.Messages))
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
	messageCount int,
) (map[library.EntryKey]library.MessageEntry, error) {
	if ref.Library == nil && messageCount > 0 {
		return nil, validate.PathedError(errors.Wrap(
			validate.ErrValidation, "select a library before choosing messages",
		), "library")
	}
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
	bases := m.Payload.FieldBases()
	known := make(set.Set[library.FieldKey], len(bases))
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

// stamp is the first field a read config stamps on an index.
type stamp struct {
	message library.MessageEntry
	field   library.BaseField
}

// checkIndexes records in stamps the index channel of each field that the i-th message
// of a read config reads from m. It returns a path-scoped validation error when an
// index already stamps another message, or a field of m that other multiplexor values
// select.
func checkIndexes(
	stamps map[channel.Key]stamp,
	m library.MessageEntry,
	i int,
	fields []ReadField,
) error {
	bases := make(map[library.FieldKey]library.BaseField)
	for _, b := range m.Payload.FieldBases() {
		bases[b.Key] = b
	}
	for j, f := range fields {
		if f.Index == 0 {
			continue
		}
		field := bases[f.Field]
		first, ok := stamps[f.Index]
		if !ok {
			stamps[f.Index] = stamp{message: m, field: field}
			continue
		}
		path := "messages." + strconv.Itoa(i) + ".fields." + strconv.Itoa(j) + ".index"
		if first.message.Key != m.Key {
			return validate.PathedError(errors.Wrapf(
				validate.ErrValidation,
				"index %d already stamps message %s",
				f.Index,
				first.message.Name,
			), path)
		}
		if !samePresence(first.field, field) {
			return validate.PathedError(errors.Wrapf(
				validate.ErrValidation,
				"fields %s and %s share index %d, but different multiplexor values "+
					"select them",
				first.field.Name,
				field.Name,
				f.Index,
			), path)
		}
	}
	return nil
}

// samePresence returns true when the same frames of a message carry a and b. Two
// fields with the same multiplexor and values share the rest of the chain above it.
func samePresence(a, b library.BaseField) bool {
	if a.Multiplexor == nil || b.Multiplexor == nil {
		return a.Multiplexor == nil && b.Multiplexor == nil
	}
	return *a.Multiplexor == *b.Multiplexor &&
		set.New(a.MultiplexValues...).Equal(set.New(b.MultiplexValues...))
}

// checkReplies returns a path-scoped validation error when an enabled polled message
// has a reply with no identifier and the config also reads a message the device sends
// unprompted, since nothing would tell the two apart.
func checkReplies(
	messages map[library.EntryKey]library.MessageEntry,
	reads []ReadMessage,
) error {
	streamed := lo.ContainsBy(reads, func(m ReadMessage) bool {
		return !m.Disabled && messages[m.Message].Query == nil
	})
	if !streamed {
		return nil
	}
	for i, m := range reads {
		entry := messages[m.Message]
		if m.Disabled || entry.Query == nil || labeled(entry.Payload) {
			continue
		}
		return validate.PathedError(errors.Wrapf(
			validate.ErrValidation,
			"message %s has a reply with no identifier, so it cannot share the task "+
				"with messages the device sends unprompted",
			entry.Name,
		), "messages."+strconv.Itoa(i)+".message")
	}
	return nil
}

// labeled returns true when a frame or line of the payload carries an identifier.
func labeled(p library.Payload) bool {
	switch v := p.Variant.(type) {
	case library.BinaryPayload:
		return v.Identifier != nil
	case library.TextPayload:
		return v.Prefix != ""
	default:
		panic(errors.Newf("unknown payload variant %T", v))
	}
}
