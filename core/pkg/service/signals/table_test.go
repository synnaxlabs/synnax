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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// receiveSamples returns the samples of the first series of the next frame.
func receiveSamples(responses confluence.Outlet[framer.StreamerResponse]) [][]byte {
	GinkgoHelper()
	var res framer.StreamerResponse
	Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
	var samples [][]byte
	for s := range res.Frame.SeriesAt(0).Samples() {
		samples = append(samples, s)
	}
	return samples
}

var _ = Describe("Table", func() {
	Describe("MarshalJSON", func() {
		It("Should marshal an entry as a variable JSON sample", func() {
			e := testUUIDEntry{
				Key:  uuid.MustParse("12345678-1234-1234-1234-123456789012"),
				Name: "test",
			}
			b := MustSucceed(signals.MarshalJSON(e))
			Expect(binary.LittleEndian.Uint32(b[:4])).To(BeEquivalentTo(len(b) - 4))
			Expect(string(b[4:])).To(ContainSubstring(`"name":"test"`))
		})
	})

	Describe("TableConfig", func() {
		It("Should name the channels after the entry type by default", func() {
			cfg := signals.DefaultTableConfig[uuid.UUID, testUUIDEntry]()
			Expect(cfg.SetName).To(Equal("sy_testuuidentry_set"))
			Expect(cfg.DeleteName).To(Equal("sy_testuuidentry_delete"))
			Expect(*cfg.SetDisabled).To(BeFalse())
		})

		It("Should fail without a function that decodes changes", func() {
			cfg := signals.DefaultTableConfig[uuid.UUID, testUUIDEntry]()
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("changes")))
		})

		It("Should fail without a function that returns the resource of a key", func() {
			cfg := signals.DefaultTableConfig[uuid.UUID, testUUIDEntry]().Override(
				signals.TableConfig[uuid.UUID, testUUIDEntry]{
					Changes: uuidTable.Changes,
				},
			)
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("resource")))
		})

		It("Should fail without a function that reads the key of a set", func() {
			cfg := signals.DefaultTableConfig[uuid.UUID, testUUIDEntry]().Override(
				signals.TableConfig[uuid.UUID, testUUIDEntry]{
					Changes:  uuidTable.Changes,
					Resource: testResource[uuid.UUID],
				},
			)
			cfg.SetKey = nil
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("set_key")))
		})

		It("Should need no set channel name when the set channel is disabled", func() {
			cfg := signals.DefaultTableConfig[uuid.UUID, testUUIDEntry]().Override(
				signals.TableConfig[uuid.UUID, testUUIDEntry]{
					Changes:     uuidTable.Changes,
					Resource:    testResource[uuid.UUID],
					SetDisabled: new(true),
				},
			)
			cfg.SetName = ""
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("PublishTable", func() {
		var sigs *signals.Provider
		BeforeEach(func() {
			sigs = MustSucceed(signals.New(signals.Config{
				DB:      db,
				Channel: channelSvc,
				Framer:  framerSvc,
			}))
		})

		It("Should write sets as JSON and UUID deletes as 16 bytes", func(
			ctx SpecContext,
		) {
			setName, deleteName := nextName(), nextName()
			Expect(sigs.PublishTable(ctx, signals.TableConfig[uuid.UUID, testUUIDEntry]{
				Changes:    uuidTable.Changes,
				Resource:   testResource[uuid.UUID],
				SetName:    setName,
				DeleteName: deleteName,
			})).To(Succeed())
			MustOpen(sigs.Start(ctx))
			set, del := retrieveChannel(ctx, setName), retrieveChannel(ctx, deleteName)
			Expect(set.DataType).To(Equal(telem.JSONT))
			Expect(del.DataType).To(Equal(telem.UUIDT))
			sets, deletes := openStreamer(ctx, set.Key()), openStreamer(ctx, del.Key())
			e := testUUIDEntry{Key: uuid.New(), Name: "published"}
			Expect(uuidTable.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
			Expect(receiveSamples(sets)).To(ConsistOf(
				WithTransform(func(b []byte) string { return string(b) },
					ContainSubstring(`"name":"published"`)),
			))
			Expect(uuidTable.NewDelete().
				Where(gorp.MatchKeys[uuid.UUID, testUUIDEntry](e.Key)).
				Exec(ctx, db)).To(Succeed())
			Expect(receiveSamples(deletes)).To(Equal([][]byte{e.Key[:]}))
		})

		It("Should write a numeric key as its raw bytes", func(ctx SpecContext) {
			deleteName := nextName()
			Expect(sigs.PublishTable(
				ctx,
				signals.TableConfig[uint32, testNumericEntry]{
					Changes:     numTable.Changes,
					Resource:    testResource[uint32],
					DeleteName:  deleteName,
					SetDisabled: new(true),
				},
			)).To(Succeed())
			MustOpen(sigs.Start(ctx))
			del := retrieveChannel(ctx, deleteName)
			Expect(del.DataType).To(Equal(telem.Uint32T))
			deletes := openStreamer(ctx, del.Key())
			e := testNumericEntry{Key: 42, Value: "numeric"}
			Expect(numTable.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
			Expect(numTable.NewDelete().
				Where(gorp.MatchKeys[uint32, testNumericEntry](42)).
				Exec(ctx, db)).To(Succeed())
			Expect(receiveSamples(deletes)).
				To(Equal([][]byte{binary.LittleEndian.AppendUint32(nil, 42)}))
		})

		It("Should write a string key as a variable sample", func(ctx SpecContext) {
			deleteName := nextName()
			Expect(sigs.PublishTable(
				ctx,
				signals.TableConfig[string, testStringEntry]{
					Changes:     stringTable.Changes,
					Resource:    testResource[string],
					DeleteName:  deleteName,
					SetDisabled: new(true),
				},
			)).To(Succeed())
			MustOpen(sigs.Start(ctx))
			del := retrieveChannel(ctx, deleteName)
			Expect(del.DataType).To(Equal(telem.StringT))
			deletes := openStreamer(ctx, del.Key())
			e := testStringEntry{Key: "entry-key", Value: 1}
			Expect(stringTable.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
			Expect(stringTable.NewDelete().
				Where(gorp.MatchKeys[string, testStringEntry]("entry-key")).
				Exec(ctx, db)).To(Succeed())
			Expect(receiveSamples(deletes)).To(Equal([][]byte{[]byte("entry-key")}))
		})

		It("Should encode sets with MarshalSet", func(ctx SpecContext) {
			setName := nextName()
			Expect(sigs.PublishTable(ctx, signals.TableConfig[uuid.UUID, testUUIDEntry]{
				Changes:    uuidTable.Changes,
				Resource:   testResource[uuid.UUID],
				SetName:    setName,
				DeleteName: nextName(),
				MarshalSet: func(e testUUIDEntry) ([]byte, error) {
					return telem.MarshalVariableSample([]byte(`"` + e.Name + `"`)), nil
				},
			})).To(Succeed())
			MustOpen(sigs.Start(ctx))
			sets := openStreamer(ctx, retrieveChannel(ctx, setName).Key())
			e := testUUIDEntry{Key: uuid.New(), Name: "custom"}
			Expect(uuidTable.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
			Expect(receiveSamples(sets)).To(Equal([][]byte{[]byte(`"custom"`)}))
		})

		It("Should create no set channel when the set channel is disabled", func(
			ctx SpecContext,
		) {
			setName := nextName()
			Expect(sigs.PublishTable(ctx, signals.TableConfig[uuid.UUID, testUUIDEntry]{
				Changes:     uuidTable.Changes,
				Resource:    testResource[uuid.UUID],
				SetName:     setName,
				DeleteName:  nextName(),
				SetDisabled: new(true),
			})).To(Succeed())
			Expect(channelSvc.NewRetrieve().
				Where(channel.MatchNames(setName)).
				Exists(ctx, db)).To(BeFalse())
		})
	})
})
