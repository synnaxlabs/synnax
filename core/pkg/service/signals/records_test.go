// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals_test

import (
	"encoding/binary"
	"encoding/json/v2"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Records", func() {
	var sigs *signals.Provider
	BeforeEach(func() {
		sigs = MustSucceed(signals.New(signals.Config{
			DB:      db,
			Channel: channelSvc,
			Framer:  framerSvc,
		}))
	})

	It("Should find the entry of each set and delete sample", func(ctx SpecContext) {
		setName, deleteName := nextName(), nextName()
		Expect(sigs.PublishTable(ctx, signals.TableConfig[uuid.UUID, testUUIDEntry]{
			Changes:    uuidTable.Changes,
			Resource:   testResource[uuid.UUID],
			SetName:    setName,
			DeleteName: deleteName,
		})).To(Succeed())
		MustOpen(sigs.Start(ctx))
		set, del := retrieveChannel(ctx, setName), retrieveChannel(ctx, deleteName)
		sets, deletes := openStreamer(ctx, set.Key()), openStreamer(ctx, del.Key())
		e := testUUIDEntry{Key: uuid.New(), Name: "record"}
		Expect(uuidTable.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
		Expect(uuidTable.NewDelete().
			Where(gorp.MatchKeys[uuid.UUID, testUUIDEntry](e.Key)).
			Exec(ctx, db)).To(Succeed())
		records := sigs.Records()
		setSample := receiveSamples(sets)[0]
		Expect(records[set.Key()](setSample)).To(Equal(testResource(e.Key)))
		deleteSample := receiveSamples(deletes)[0]
		Expect(records[del.Key()](deleteSample)).To(Equal(testResource(e.Key)))
	})

	It("Should find the entry of a numeric key delete sample", func(ctx SpecContext) {
		deleteName := nextName()
		Expect(sigs.PublishTable(ctx, signals.TableConfig[uint32, testNumericEntry]{
			Changes:     numTable.Changes,
			Resource:    testResource[uint32],
			DeleteName:  deleteName,
			SetDisabled: new(true),
		})).To(Succeed())
		recordOf := sigs.Records()[retrieveChannel(ctx, deleteName).Key()]
		Expect(recordOf(binary.LittleEndian.AppendUint32(nil, 42))).
			To(Equal(testResource(uint32(42))))
	})

	It("Should find the entry of a string key delete sample", func(ctx SpecContext) {
		deleteName := nextName()
		Expect(sigs.PublishTable(ctx, signals.TableConfig[string, testStringEntry]{
			Changes:     stringTable.Changes,
			Resource:    testResource[string],
			DeleteName:  deleteName,
			SetDisabled: new(true),
		})).To(Succeed())
		recordOf := sigs.Records()[retrieveChannel(ctx, deleteName).Key()]
		Expect(recordOf([]byte("entry-key"))).To(Equal(testResource("entry-key")))
	})

	It("Should read the key of a set sample with SetKey", func(ctx SpecContext) {
		setName, key := nextName(), uuid.New()
		Expect(sigs.PublishTable(ctx, signals.TableConfig[uuid.UUID, testUUIDEntry]{
			Changes:    uuidTable.Changes,
			Resource:   testResource[uuid.UUID],
			SetKey:     func([]byte) (uuid.UUID, error) { return key, nil },
			SetName:    setName,
			DeleteName: nextName(),
		})).To(Succeed())
		recordOf := sigs.Records()[retrieveChannel(ctx, setName).Key()]
		Expect(recordOf([]byte(`"custom"`))).To(Equal(testResource(key)))
	})

	Describe("Set samples", func() {
		var setName string
		BeforeEach(func(ctx SpecContext) {
			setName = nextName()
			Expect(sigs.PublishTable(
				ctx,
				signals.TableConfig[uuid.UUID, testUUIDEntry]{
					Changes:    uuidTable.Changes,
					Resource:   testResource[uuid.UUID],
					SetName:    setName,
					DeleteName: nextName(),
				},
			)).To(Succeed())
		})

		It("Should fail on a sample with no key", func(ctx SpecContext) {
			recordOf := sigs.Records()[retrieveChannel(ctx, setName).Key()]
			Expect(recordOf([]byte(`{"name":"keyless"}`))).Error().
				To(MatchError(ContainSubstring("sample has no key")))
		})

		It("Should fail on a sample that is not JSON", func(ctx SpecContext) {
			recordOf := sigs.Records()[retrieveChannel(ctx, setName).Key()]
			Expect(recordOf([]byte("not json"))).Error().
				To(MatchError(ContainSubstring("invalid character")))
		})
	})

	It("Should find the entry that an action batch edits", func(ctx SpecContext) {
		name := nextName()
		Expect(sigs.PublishActions(ctx, signals.ActionConfig[uuid.UUID, action]{
			Name:     name,
			Actions:  actions.NewState[uuid.UUID, action](db),
			Resource: testResource[uuid.UUID],
		})).To(Succeed())
		key := uuid.New()
		sample := MustSucceed(json.Marshal(scoped{
			Key:     key,
			Actions: []action{{Type: "rename", Payload: "next"}},
		}))
		recordOf := sigs.Records()[retrieveChannel(ctx, name).Key()]
		Expect(recordOf(sample)).To(Equal(testResource(key)))
	})

	It("Should hold no channel of a JSON publisher", func(ctx SpecContext) {
		MustOpen(sigs.PublishJSON(ctx, signals.JSONPublisherConfig[string]{
			Observable: observe.New[string](),
			SetName:    nextName(),
		}))
		Expect(sigs.Records()).To(BeEmpty())
	})
})

var _ = Describe("UnmarshalEntryKey", func() {
	It("Should return the key of the entry that a JSON sample holds", func() {
		e := testStringEntry{Key: "entry", Value: 3}
		sample := MustSucceed(json.Marshal(e))
		Expect(signals.UnmarshalEntryKey[string, testStringEntry](sample)).
			To(Equal("entry"))
	})

	It("Should fail on a sample that is not JSON", func() {
		Expect(signals.UnmarshalEntryKey[string, testStringEntry]([]byte("{"))).
			Error().To(MatchError(ContainSubstring("unexpected EOF")))
	})
})
