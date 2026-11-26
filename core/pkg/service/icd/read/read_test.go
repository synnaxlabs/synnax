// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package read_test

import (
	"context"
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func jsonField(name, pointer string, v icd.JSONFieldVariant) icd.JSONField {
	f := icd.JSONField{Variant: v}
	f.SetBase(
		icd.JSONBaseField{Key: uuid.New(), Name: name, Scale: 1, Pointer: pointer},
	)
	return f
}

func jsonMessage(name string, fields ...icd.JSONField) icd.Message {
	return icd.Message{
		Key:     uuid.New(),
		Name:    name,
		Payload: icd.Payload{Variant: icd.JSONPayload{Fields: fields}},
	}
}

var (
	ts = jsonField(
		"ts",
		"/ts",
		icd.TimestampJSONField{TimeFormat: payload.TimeFormatUnixSec},
	)
	temp  = jsonField("temp", "/temp", icd.FloatJSONField{})
	state = jsonField("state", "/state", icd.StringJSONField{})
	raw   = jsonField("raw", "", icd.BytesJSONField{})
	// flat is a float field with a zero scale.
	flat = icd.JSONField{Variant: icd.FloatJSONField{JSONBaseField: icd.JSONBaseField{
		Key: uuid.New(), Name: "flat", Pointer: "/flat",
	}}}
	// count is a scaled integer field.
	count = icd.JSONField{
		Variant: icd.IntegerJSONField{JSONBaseField: icd.JSONBaseField{
			Key: uuid.New(), Name: "count", Pointer: "/count", Scale: 2,
		}},
	}
	stamp = telem.SecondTS
	// defaulted is a timestamp field that takes a default.
	defaulted = jsonField("ts", "/ts", icd.TimestampJSONField{
		TimeFormat: payload.TimeFormatUnixSec,
		Default: &icd.TimestampDefault{
			Variant: icd.ConstantTimestampDefault{Value: stamp},
		},
	})
	status  = jsonMessage("Status", ts, temp, state, raw, flat, count)
	stamped = jsonMessage("Stamped", defaulted)
	frame   = icd.Message{
		Key:     uuid.New(),
		Name:    "Frame",
		Payload: icd.Payload{Variant: icd.BinaryPayload{}},
	}
	plant = icd.ICD{
		Key:      uuid.New(),
		Name:     "Plant",
		Messages: []icd.Message{status, stamped, frame},
	}
)

const (
	timeKey  channel.Key = 1
	valueKey channel.Key = 2
	stampKey channel.Key = 3
	levelKey channel.Key = 4
	// pressureKey is a data channel on an index channel other than timeKey.
	pressureKey channel.Key = 5
	countKey    channel.Key = 7
)

// held holds the channels that the channel lookup of the specs finds.
var held = map[channel.Key]icdv0.Channel{
	timeKey: {
		Name:     "time",
		DataType: telem.TimestampT,
		Index:    true,
		IndexKey: timeKey,
	},
	valueKey:    {Name: "value", DataType: telem.Float64T, IndexKey: timeKey},
	stampKey:    {Name: "stamp", DataType: telem.TimestampT},
	levelKey:    {Name: "level", DataType: telem.Float64T, IndexKey: timeKey},
	pressureKey: {Name: "pressure", DataType: telem.Float64T, IndexKey: 6},
	countKey:    {Name: "count", DataType: telem.Int64T, IndexKey: timeKey},
}

func bind(f icd.JSONField, key channel.Key) icd.ReadBinding {
	return icd.ReadBinding{Field: f.Base().Key, Channel: key}
}

func readOf(m icd.Message, bindings ...icd.ReadBinding) icd.ReadMessage {
	return icd.ReadMessage{Message: m.Key, Bindings: bindings}
}

func indexed(f icd.JSONField, rm icd.ReadMessage) icd.ReadMessage {
	key := f.Base().Key
	rm.Index = &key
	return rm
}

// readAt returns the read of rms of the ICD with key, at reads.i of a config.
func readAt(i int, key icd.Key, rms ...icd.ReadMessage) read.Read {
	return read.Read{Path: []string{"reads", strconv.Itoa(i)}, ICD: key, Messages: rms}
}

var _ = Describe("Read", func() {
	var (
		resolver read.Resolver
		// asked holds the keys of each channel lookup.
		asked [][]channel.Key
	)
	BeforeEach(func() {
		asked = nil
		resolver = read.NewResolver(icdSvc, func(
			_ context.Context,
			_ gorp.Snapshot,
			keys []channel.Key,
		) (map[channel.Key]icdv0.Channel, error) {
			asked = append(asked, keys)
			return held, nil
		})
	})

	// checker returns the Checker of reads, which is given plant, after it reads each
	// of them, and the error of the first read that it refuses.
	checker := func(ctx context.Context, reads ...read.Read) (read.Checker, error) {
		GinkgoHelper()
		c := MustSucceed(resolver.NewChecker(
			ctx, db, reads, nil, map[icd.Key]icd.ICD{plant.Key: plant},
		))
		for _, rd := range reads {
			if err := c.Read(rd); err != nil {
				return read.Checker{}, err
			}
		}
		return c, nil
	}

	// check checks reads with the Checker of them.
	check := func(ctx context.Context, reads ...read.Read) error {
		GinkgoHelper()
		_, err := checker(ctx, reads...)
		return err
	}

	Describe("Checker", func() {
		Describe("Read", func() {
			It("Should accept a read whose fields suit their channels", func(
				ctx SpecContext,
			) {
				rm := indexed(
					ts,
					readOf(status, bind(ts, timeKey), bind(temp, valueKey)),
				)
				Expect(check(ctx, readAt(0, plant.Key, rm))).To(Succeed())
			})

			It("Should accept a message with no index field on one index channel", func(
				ctx SpecContext,
			) {
				rm := readOf(status, bind(temp, valueKey), bind(temp, levelKey))
				Expect(check(ctx, readAt(0, plant.Key, rm))).To(Succeed())
			})

			It("Should accept a message whose channels are on two index channels", func(
				ctx SpecContext,
			) {
				rm := readOf(status, bind(temp, valueKey), bind(temp, pressureKey))
				Expect(check(ctx, readAt(0, plant.Key, rm))).To(Succeed())
			})

			It("Should accept a scaled field on a float channel", func(
				ctx SpecContext,
			) {
				rm := readOf(status, bind(count, valueKey))
				Expect(check(ctx, readAt(0, plant.Key, rm))).To(Succeed())
			})

			It("Should accept a binding to a channel that the lookup does not find",
				func(ctx SpecContext) {
					rm := readOf(status, bind(state, 99))
					Expect(check(ctx, readAt(0, plant.Key, rm))).To(Succeed())
				},
			)

			It("Should accept a data type that the field converts to", func(
				ctx SpecContext,
			) {
				rm := readOf(status, icd.ReadBinding{
					Field: temp.Base().Key, DataType: telem.Float32T,
				})
				Expect(check(ctx, readAt(0, plant.Key, rm))).To(Succeed())
			})

			DescribeTable("Should reject a read with the path of the problem",
				func(ctx SpecContext, rm icd.ReadMessage, path, msg string) {
					err := check(ctx, readAt(0, plant.Key, rm))
					Expect(err).To(MatchError(validate.ErrValidation))
					Expect(err).To(MatchError(HavePrefix(path + ": ")))
					Expect(err).To(MatchError(ContainSubstring(msg)))
				},
				Entry("a message that the ICD lacks",
					icd.ReadMessage{Message: uuid.New()},
					"reads.0.messages.0.message", "does not exist in ICD Plant",
				),
				Entry("a message that is not JSON",
					readOf(frame),
					"reads.0.messages.0.message",
					"message Frame is not a JSON message",
				),
				Entry("a field that the message lacks",
					readOf(status, icd.ReadBinding{Field: uuid.New()}),
					"reads.0.messages.0.bindings.0.field",
					"does not exist in message Status",
				),
				Entry(
					"a bytes field at the root",
					readOf(status, icd.ReadBinding{Field: raw.Base().Key}),
					"reads.0.messages.0.bindings.0.field",
					"bytes field raw requires a JSON pointer",
				),
				Entry("a field with a zero scale",
					readOf(status, bind(flat, valueKey)),
					"reads.0.messages.0.bindings.0.field",
					"field flat: scale must be non-zero",
				),
				Entry("a data type that the field kind does not suit",
					readOf(status, icd.ReadBinding{
						Field: state.Base().Key, DataType: telem.Float64T,
					}),
					"reads.0.messages.0.bindings.0.data_type",
					"field state of kind string cannot read into data type float64",
				),
				Entry("a channel that the field kind does not convert to",
					readOf(status, bind(state, valueKey)),
					"reads.0.messages.0.bindings.0.channel",
					"field state of kind string cannot read into channel value of "+
						"data type float64",
				),
				Entry("a scaled field with an integer data type",
					readOf(status, icd.ReadBinding{
						Field: count.Base().Key, DataType: telem.Int64T,
					}),
					"reads.0.messages.0.bindings.0.data_type",
					"scaled field count of kind integer cannot read into data type "+
						"int64",
				),
				Entry(
					"a scaled field on an integer channel",
					readOf(status, bind(count, countKey)),
					"reads.0.messages.0.bindings.0.channel",
					"scaled field count of kind integer cannot read into channel count "+
						"of data type int64",
				),
				Entry("an index field that is not a timestamp",
					indexed(temp, readOf(status, bind(temp, valueKey))),
					"reads.0.messages.0.index",
					"the index field temp must be a timestamp field",
				),
				Entry("an index field with a default",
					indexed(defaulted, readOf(stamped, bind(defaulted, timeKey))),
					"reads.0.messages.0.index",
					"the index field ts cannot take a default",
				),
				Entry("an index field with no binding",
					indexed(ts, readOf(status, bind(temp, valueKey))),
					"reads.0.messages.0.index", "the index field ts has no binding",
				),
				Entry("an index that names no field",
					indexed(jsonField("gone", "/gone", icd.TimestampJSONField{}),
						readOf(status, bind(temp, valueKey))),
					"reads.0.messages.0.index", "does not exist in message Status",
				),
				Entry("an index field on a data channel",
					indexed(ts, readOf(status, bind(ts, stampKey))),
					"reads.0.messages.0.bindings.0.channel",
					"the index field ts must write to an index channel",
				),
				Entry("a data field on an index channel",
					readOf(status, bind(ts, timeKey)),
					"reads.0.messages.0.bindings.0.channel",
					"field ts cannot write to index channel time, because only the "+
						"index field can. Bind field ts to a data channel",
				),
			)

			It("Should reject a channel that an earlier read writes", func(
				ctx SpecContext,
			) {
				rm := readOf(status, bind(temp, valueKey))
				Expect(check(
					ctx, readAt(0, plant.Key, rm), readAt(1, plant.Key, rm),
				)).To(MatchError(
					"reads.1.messages.0.bindings.0.channel: channel 2 is also " +
						"written by reads.0.messages.0.bindings.0.channel",
				))
			})
		})

		Describe("ClaimIndexes", func() {
			// claimIndexes claims the index channels of reads with the Checker of them,
			// which accepts each of them first.
			claimIndexes := func(ctx context.Context, reads ...read.Read) error {
				GinkgoHelper()
				c := MustSucceed(checker(ctx, reads...))
				for _, rd := range reads {
					if err := c.ClaimIndexes(rd); err != nil {
						return err
					}
				}
				return nil
			}

			It("Should take an index channel that the index field binds", func(
				ctx SpecContext,
			) {
				rm := indexed(
					ts,
					readOf(status, bind(ts, timeKey), bind(temp, valueKey)),
				)
				Expect(claimIndexes(ctx, readAt(0, plant.Key, rm))).To(Succeed())
			})

			It("Should take an index channel of two bound channels once", func(
				ctx SpecContext,
			) {
				rm := readOf(status, bind(temp, valueKey), bind(temp, levelKey))
				Expect(claimIndexes(ctx, readAt(0, plant.Key, rm))).To(Succeed())
			})

			It("Should reject a message that writes the index channel of another", func(
				ctx SpecContext,
			) {
				rd := readAt(
					0,
					plant.Key,
					indexed(
						ts,
						readOf(status, bind(ts, timeKey), bind(temp, valueKey)),
					),
					readOf(status, bind(temp, levelKey)),
				)
				Expect(claimIndexes(ctx, rd)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError("reads.0.messages.1: channel 1 is also written by "+
						"reads.0.messages.0.bindings.0.channel"),
				))
			})

			DescribeTable("Should reject two messages that write one index channel",
				func(ctx SpecContext, first icd.ReadMessage, other string) {
					Expect(claimIndexes(
						ctx,
						readAt(0, plant.Key, first),
						readAt(1, plant.Key, readOf(status, bind(temp, levelKey))),
					)).To(MatchError(
						"reads.1.messages.0: channel 1 is also written by " + other,
					))
				},
				Entry("when neither has an index field",
					readOf(status, bind(temp, valueKey)),
					"reads.0.messages.0",
				),
				Entry(
					"when the other binds an index field to it",
					indexed(
						ts,
						readOf(status, bind(ts, timeKey), bind(temp, valueKey)),
					),
					"reads.0.messages.0.bindings.0.channel",
				),
				Entry("when the other writes it among two index channels",
					readOf(status, bind(temp, pressureKey), bind(temp, valueKey)),
					"reads.0.messages.0",
				),
			)
		})

		Describe("Tag", func() {
			// takes is the check of a tag that takes floats.
			takes := func(dt telem.DataType) bool { return dt == telem.Float64T }

			It("Should take a channel that can take the values of the tag", func(
				ctx SpecContext,
			) {
				tags := []channel.Key{valueKey}
				c := MustSucceed(resolver.NewChecker(ctx, db, nil, tags, nil))
				Expect(c.Tag(valueKey, []string{"a", "channel"}, takes)).To(Succeed())
				Expect(c.Tag(valueKey, []string{"b", "channel"}, takes)).To(MatchError(
					"b.channel: channel 2 is also written by a.channel",
				))
			})

			It("Should take a channel that the lookup does not find", func(
				ctx SpecContext,
			) {
				c := MustSucceed(resolver.NewChecker(ctx, db, nil, nil, nil))
				Expect(c.Tag(99, []string{"a", "channel"}, takes)).To(Succeed())
			})

			It("Should reject an index channel", func(ctx SpecContext) {
				tags := []channel.Key{timeKey}
				c := MustSucceed(resolver.NewChecker(ctx, db, nil, tags, nil))
				err := c.Tag(timeKey, []string{"a", "channel"}, takes)
				Expect(err).To(MatchError(validate.ErrValidation))
				Expect(err).To(MatchError(
					"a.channel: channel time is an index channel",
				))
			})

			It("Should reject a channel that cannot take the values of the tag", func(
				ctx SpecContext,
			) {
				tags := []channel.Key{countKey}
				c := MustSucceed(resolver.NewChecker(ctx, db, nil, tags, nil))
				err := c.Tag(countKey, []string{"a", "channel"}, takes)
				Expect(err).To(MatchError(validate.ErrValidation))
				Expect(err).To(MatchError(
					"a.channel: channel count of data type int64 cannot take the " +
						"values of a tag",
				))
			})

			It("Should reject a channel that another binding or tag writes", func(
				ctx SpecContext,
			) {
				rd := readAt(0, plant.Key, readOf(status, bind(temp, valueKey)))
				c := MustSucceed(checker(ctx, rd))
				Expect(c.Tag(valueKey, []string{"tag"}, takes)).To(MatchError(
					"tag: channel 2 is also written by " +
						"reads.0.messages.0.bindings.0.channel",
				))
			})
		})

		Describe("ClaimIndex", func() {
			It("Should reject a tag whose index channel another tag writes", func(
				ctx SpecContext,
			) {
				takes := func(telem.DataType) bool { return true }
				tags := []channel.Key{valueKey, levelKey}
				c := MustSucceed(resolver.NewChecker(ctx, db, nil, tags, nil))
				Expect(c.Tag(valueKey, []string{"a", "channel"}, takes)).To(Succeed())
				Expect(c.Tag(levelKey, []string{"b", "channel"}, takes)).To(Succeed())
				Expect(c.ClaimIndex(valueKey, []string{"a", "channel"})).To(Succeed())
				Expect(c.ClaimIndex(levelKey, []string{"b", "channel"})).To(MatchError(
					"b.channel: channel 1 is also written by a.channel",
				))
			})
		})
	})

	Describe("NewChecker", func() {
		It("Should look up the channel of each tag with the channels of reads", func(
			ctx SpecContext,
		) {
			rd := readAt(0, plant.Key, readOf(status, bind(temp, valueKey)))
			Expect(resolver.NewChecker(
				ctx, db, []read.Read{rd}, []channel.Key{levelKey, valueKey}, nil,
			)).Error().ToNot(HaveOccurred())
			Expect(asked).To(HaveExactElements(ConsistOf(levelKey, valueKey)))
		})

		It("Should retrieve an ICD that the given ICDs lack", func(ctx SpecContext) {
			stored := icd.ICD{Name: "Stored", Messages: []icd.Message{status}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Create(ctx, &stored)
			})).To(Succeed())
			Expect(check(ctx, readAt(
				0, stored.Key, icd.ReadMessage{Message: uuid.New()},
			))).To(MatchError(ContainSubstring("does not exist in ICD Stored")))
		})

		It("Should skip a read whose ICD does not exist", func(ctx SpecContext) {
			Expect(check(ctx, readAt(
				0, uuid.New(), icd.ReadMessage{Message: uuid.New()},
			))).To(Succeed())
		})

		It("Should look up each channel that a binding names once", func(
			ctx SpecContext,
		) {
			Expect(check(
				ctx,
				readAt(0, plant.Key, readOf(status, bind(temp, valueKey))),
				readAt(1, plant.Key, readOf(
					stamped,
					bind(defaulted, stampKey),
					icd.ReadBinding{Field: defaulted.Base().Key},
				)),
				readAt(2, uuid.New(), readOf(status, bind(temp, valueKey))),
			)).To(Succeed())
			Expect(asked).To(HaveExactElements(ConsistOf(valueKey, stampKey)))
		})

		It("Should return the error of the channel lookup", func(ctx SpecContext) {
			resolver = read.NewResolver(icdSvc, func(
				context.Context,
				gorp.Snapshot,
				[]channel.Key,
			) (map[channel.Key]icdv0.Channel, error) {
				return nil, errors.New("no channels")
			})
			rd := readAt(0, plant.Key, readOf(status, bind(temp, valueKey)))
			Expect(resolver.NewChecker(ctx, db, []read.Read{rd}, nil, nil)).Error().
				To(MatchError("no channels"))
		})
	})

	Describe("Hash", func() {
		var ic icd.ICD
		BeforeEach(func(ctx SpecContext) {
			ic = icd.ICD{Name: "Plant", Messages: []icd.Message{status, stamped}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Create(ctx, &ic)
			})).To(Succeed())
		})
		sum := func(ctx context.Context, reads ...read.Read) string {
			GinkgoHelper()
			h := content.NewHash()
			Expect(resolver.Hash(ctx, db, &h, reads)).To(Succeed())
			return h.Sum()
		}
		// setPointer sets the pointer of f, a field of m, to pointer.
		setPointer := func(
			ctx context.Context,
			m icd.Message,
			f icd.JSONField,
			pointer string,
		) {
			GinkgoHelper()
			path := []string{
				"messages", m.Key.String(),
				"payload", "fields", f.Base().Key.String(),
				"pointer",
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Apply(ctx, ic.Key, icd.NewSetAction(
					icd.SetPayload{Path: path, Value: pointer},
				))
			})).To(Succeed())
		}

		It("Should give the same sum for the same reads", func(ctx SpecContext) {
			rd := readAt(0, ic.Key, readOf(status, bind(temp, valueKey)))
			Expect(sum(ctx, rd)).To(Equal(sum(ctx, rd)))
		})

		It("Should change when the payload of a read message changes", func(
			ctx SpecContext,
		) {
			rd := readAt(0, ic.Key, readOf(status, bind(temp, valueKey)))
			before := sum(ctx, rd)
			setPointer(ctx, status, temp, "/temperature")
			Expect(sum(ctx, rd)).ToNot(Equal(before))
		})

		It("Should not change when a message that no read reads changes", func(
			ctx SpecContext,
		) {
			rd := readAt(0, ic.Key, readOf(status, bind(temp, valueKey)))
			before := sum(ctx, rd)
			setPointer(ctx, stamped, defaulted, "/stamp")
			Expect(sum(ctx, rd)).To(Equal(before))
		})

		It("Should mark a read of a message that the ICD lacks", func(
			ctx SpecContext,
		) {
			lacked := readAt(0, ic.Key, icd.ReadMessage{Message: uuid.New()})
			Expect(sum(ctx, lacked)).ToNot(Equal(sum(ctx, readAt(0, ic.Key))))
		})

		It("Should fail for an ICD that does not exist", func(ctx SpecContext) {
			h := content.NewHash()
			Expect(resolver.Hash(ctx, db, &h, []read.Read{readAt(0, uuid.New())})).
				To(MatchError(query.ErrNotFound))
		})
	})
})
