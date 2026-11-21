// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	icdv1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/kafka"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
)

// jsonField returns a JSON field of an ICD message.
func jsonField(name, pointer string, v icd.JSONFieldVariant) icd.JSONField {
	base := icd.JSONBaseField{Key: uuid.New(), Name: name, Scale: 1, Pointer: pointer}
	f := icd.JSONField{Variant: v}
	f.SetBase(base)
	return f
}

// jsonMessage returns a JSON message of an ICD, read when a record value meets match.
func jsonMessage(
	name string,
	match *icd.JSONMatch,
	fields ...icd.JSONField,
) icd.Message {
	return icd.Message{
		Key:     uuid.New(),
		Name:    name,
		Payload: icd.Payload{Variant: icd.JSONPayload{Match: match, Fields: fields}},
	}
}

// createICD creates an ICD that holds messages.
func createICD(ctx context.Context, messages ...icd.Message) icd.ICD {
	GinkgoHelper()
	ic := icd.ICD{Name: "Plant", Messages: messages}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return icdSvc.NewWriter(tx).Create(ctx, &ic)
	})).To(Succeed())
	return ic
}

// jsonFields returns the fields of m, which must have a JSON payload.
func jsonFields(m icd.Message) []icd.JSONField {
	GinkgoHelper()
	p, ok := m.Payload.Variant.(icd.JSONPayload)
	Expect(ok).To(BeTrue())
	return p.Fields
}

// fieldKey returns the key of field i of m.
func fieldKey(m icd.Message, i int) icd.FieldKey {
	GinkgoHelper()
	return jsonFields(m)[i].Base().Key
}

// bind returns the binding that writes field i of m to ch.
func bind(m icd.Message, i int, ch channel.Channel) icd.ReadBinding {
	return icd.ReadBinding{Field: fieldKey(m, i), Channel: ch.Key()}
}

// lift moves records, which hold their fields inline, into a new ICD with one message
// for each record, as a write of a config from before ICDs does.
func lift(ctx context.Context, records ...kafka.InlineRecord) []kafka.ReadRecord {
	GinkgoHelper()
	cfg := kafka.ReadConfig{Inline: &kafka.InlineRead{Records: records}}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		old, _, err := v1.ResolveReadConfig(
			ctx, tx, channels, "Plant", &cfg, func(string, string) {},
		)
		if err != nil {
			return err
		}
		ic, err := icdv1.MigrateICD(ctx, old)
		if err != nil {
			return err
		}
		return icdSvc.NewWriter(tx).Create(ctx, &ic)
	})).To(Succeed())
	return cfg.Topics[0].Records
}

// oneTopic returns the topics of a task that reads records from the topic name.
func oneTopic(name string, records []kafka.ReadRecord) []kafka.ReadTopic {
	return []kafka.ReadTopic{{Name: name, Records: records}}
}

// channels returns the channels among keys, read through snap.
func channels(
	ctx context.Context,
	snap gorp.Snapshot,
	keys []channel.Key,
) (map[channel.Key]v1.Channel, error) {
	var found []channel.Channel
	if err := channelSvc.NewRetrieve().
		Where(channel.MatchKeys(keys...)).
		SkipMissingKeys().
		Entries(&found).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	return lo.SliceToMap(found, func(c channel.Channel) (channel.Key, v1.Channel) {
		return c.Key(), v1.Channel{Name: c.Name, DataType: c.DataType, Index: c.IsIndex}
	}), nil
}
