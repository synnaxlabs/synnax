// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Log", func(ctx SpecContext) {
			l := log.Log{
				Name: "test",
				Channels: []log.ChannelEntry{
					{Channel: channel.Key(1), Color: color.MustFromHex("#ff0000")},
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(l.Key).ToNot(Equal(uuid.Nil))
		})

		It("Should establish a ParentOf relationship to the project", func(ctx SpecContext) {
			l := log.Log{Name: "with-ws"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(otg.NewWriter(tx).HasRelationship(
				ctx,
				project.OntologyID(ws.Key),
				ontology.RelationshipTypeParentOf,
				log.OntologyID(l.Key),
			)).To(BeTrue())
		})

		It("Should skip the project ParentOf relationship when ws is uuid.Nil", func(ctx SpecContext) {
			l := log.Log{Name: "no-ws"}
			Expect(svc.NewWriter(tx).Create(ctx, uuid.Nil, &l)).To(Succeed())
			Expect(otg.NewWriter(tx).HasRelationship(
				ctx,
				project.OntologyID(ws.Key),
				ontology.RelationshipTypeParentOf,
				log.OntologyID(l.Key),
			)).To(BeFalse())
		})

		It("Should still register the resource in the ontology when ws is uuid.Nil", func(ctx SpecContext) {
			l := log.Log{Name: "orphan"}
			Expect(svc.NewWriter(tx).Create(ctx, uuid.Nil, &l)).To(Succeed())
			var resource ontology.Resource
			Expect(otg.NewRetrieve().
				WhereIDs(log.OntologyID(l.Key)).
				Entry(&resource).
				Exec(ctx, tx)).To(Succeed())
			Expect(resource.ID).To(Equal(log.OntologyID(l.Key)))
		})

		It("Should update the existing log when called with an existing key", func(ctx SpecContext) {
			key := uuid.New()
			first := log.Log{Key: key, Name: "first"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &first)).To(Succeed())
			second := log.Log{Key: key, Name: "second"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &second)).To(Succeed())
			var res log.Log
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("second"))
		})
	})
	Describe("Update", func() {
		It("Should rename a Log", func(ctx SpecContext) {
			l := log.Log{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, l.Key, "test2")).To(Succeed())
			var res log.Log
			Expect(svc.NewRetrieve().Where(log.MatchKeys(l.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetData", func() {
		It("Should replace every body field on the Log while preserving Key and Name", func(ctx SpecContext) {
			l := log.Log{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(svc.NewWriter(tx).SetData(ctx, l.Key, log.Log{
				Name: "ignored-name",
				Channels: []log.ChannelEntry{
					{Channel: channel.Key(1), Color: color.MustFromHex("#ff0000")},
				},
				TimestampPrecision: 2,
			})).To(Succeed())
			var res log.Log
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(l.Key)).
				Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test"))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.TimestampPrecision).To(Equal(int32(2)))
		})
	})
	Describe("Delete", func() {
		It("Should delete a Log so it is no longer retrievable", func(ctx SpecContext) {
			l := log.Log{Name: "to-delete"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, l.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(l.Key)).
				Entry(&log.Log{}).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should remove the log's ontology resource", func(ctx SpecContext) {
			l := log.Log{Name: "to-delete-otg"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, l.Key)).To(Succeed())
			Expect(otg.NewRetrieve().
				WhereIDs(log.OntologyID(l.Key)).
				Entry(&ontology.Resource{}).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple logs in one call", func(ctx SpecContext) {
			a := log.Log{Name: "a"}
			b := log.Log{Name: "b"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &a)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &b)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, a.Key, b.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(log.MatchKeys(a.Key, b.Key)).
				Entries(&[]log.Log{}).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})
})
