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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// ResolveRead stamps cfg from the ICD it references in snap. It returns a path-scoped
// validation error when cfg has messages but no ICD, when a message or field it reads
// is not in its ICD, when fields that arrive at different times share an index, or
// when a polled reply with no identifier could be confused with a message the device
// sends unprompted.
func ResolveRead(ctx context.Context, snap gorp.Snapshot, cfg *ReadConfig) error {
	keys := lo.Map(cfg.Messages, func(m ReadMessage, _ int) icd.MessageKey {
		return m.Message
	})
	messages, err := stampConfig(ctx, snap, &cfg.Reference, keys)
	if err != nil {
		return err
	}
	stamps := make(map[channel.Key]stamp)
	for i, m := range cfg.Messages {
		fields := lo.Map(m.Fields, func(f ReadField, _ int) icd.FieldKey {
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

// ResolveWrite stamps cfg from the ICD it references in snap. It returns a path-scoped
// validation error when cfg has messages but no ICD, or when a message or field it
// sends is not in its ICD.
func ResolveWrite(ctx context.Context, snap gorp.Snapshot, cfg *WriteConfig) error {
	keys := lo.Map(cfg.Messages, func(m WriteMessage, _ int) icd.MessageKey {
		return m.Message
	})
	messages, err := stampConfig(ctx, snap, &cfg.Reference, keys)
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

func stampConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	ref *icd.Reference,
	keys []icd.MessageKey,
) (map[icd.MessageKey]icd.Message, error) {
	if ref.ICD == nil && len(keys) > 0 {
		return nil, validate.PathedError(errors.Wrap(
			validate.ErrValidation, "select an ICD before choosing messages",
		), "icd")
	}
	ic, err := icd.Stamp(ctx, snap, ref, keys)
	if err != nil {
		return nil, err
	}
	return lo.KeyBy(
		ic.Messages,
		func(m icd.Message) icd.MessageKey { return m.Key },
	), nil
}

func checkMessage(
	messages map[icd.MessageKey]icd.Message,
	index int,
	key icd.MessageKey,
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

// stamp is the first field a read config stamps on an index.
type stamp struct {
	message icd.Message
	field   icd.BaseField
}

// checkIndexes records in stamps the index channel of each field that the i-th message
// of a read config reads from m. It returns a path-scoped validation error when an
// index already stamps another message, or a field of m that other multiplexor values
// select.
func checkIndexes(
	stamps map[channel.Key]stamp,
	m icd.Message,
	i int,
	fields []ReadField,
) error {
	bases := make(map[icd.FieldKey]icd.BaseField)
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
func samePresence(a, b icd.BaseField) bool {
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
	messages map[icd.MessageKey]icd.Message,
	reads []ReadMessage,
) error {
	streamed := lo.ContainsBy(reads, func(m ReadMessage) bool {
		return !m.Disabled && messages[m.Message].Query == nil
	})
	if !streamed {
		return nil
	}
	for i, m := range reads {
		msg := messages[m.Message]
		if m.Disabled || msg.Query == nil || labeled(msg.Payload) {
			continue
		}
		return validate.PathedError(errors.Wrapf(
			validate.ErrValidation,
			"message %s has a reply with no identifier, so it cannot share the task "+
				"with messages the device sends unprompted",
			msg.Name,
		), "messages."+strconv.Itoa(i)+".message")
	}
	return nil
}

// labeled returns true when a frame or line of the payload carries an identifier.
func labeled(p icd.Payload) bool {
	switch v := p.Variant.(type) {
	case icd.BinaryPayload:
		return v.Identifier != nil
	case icd.TextPayload:
		return v.Prefix != ""
	default:
		panic(errors.Newf("unknown payload variant %T", v))
	}
}
