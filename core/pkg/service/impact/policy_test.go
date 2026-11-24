// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

type policyChannel struct{ Key uint32 }

func (c policyChannel) GorpKey() uint32 { return c.Key }

type policyHolder struct {
	Key     string
	Channel uint32
}

func (h policyHolder) GorpKey() string { return h.Key }

type policyKeeper struct {
	Key     string
	Channel uint32
}

func (h policyKeeper) GorpKey() string { return h.Key }

type policyTask struct {
	Key     string
	Channel uint32
}

func (h policyTask) GorpKey() string { return h.Key }

func holderReference[E gorp.Entry[string]](
	channelOf func(*E) *uint32,
	retargets bool,
) []gorp.Reference[string, E] {
	cfg := gorp.KeyReferenceConfig[string, E, uint32]{
		Name:     "channel",
		Table:    "channel",
		OnDelete: gorp.Detach,
		Kind:     gorp.Use,
		Targets:  func(e *E) []uint32 { return []uint32{*channelOf(e)} },
		Detach:   func(e *E, _ uint32) { *channelOf(e) = 0 },
	}
	if retargets {
		cfg.Retarget = func(e *E, to map[uint32]uint32) {
			*channelOf(e) = to[*channelOf(e)]
		}
	}
	return []gorp.Reference[string, E]{gorp.NewKeyReference(cfg)}
}

var _ = Describe("Policies", Ordered, func() {
	var db *gorp.DB
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, policyChannel]{
			DB:   db,
			Name: "channel",
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, policyHolder]{
			DB:   db,
			Name: "display",
			References: holderReference(func(h *policyHolder) *uint32 {
				return &h.Channel
			}, true),
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, policyKeeper]{
			DB:   db,
			Name: "keeper",
			References: holderReference(func(h *policyKeeper) *uint32 {
				return &h.Channel
			}, false),
		}))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, policyTask]{
			DB:   db,
			Name: "read_config",
			References: holderReference(func(h *policyTask) *uint32 {
				return &h.Channel
			}, false),
		}))
	})
	open := func(links map[gorp.ReferenceID]impact.Policy) error {
		_, err := impact.OpenService(impact.ServiceConfig{
			Channel:  &channel.Service{},
			Group:    &group.Service{},
			RecordOf: func(gorp.Target) resource.ID { return resource.ID{} },
			Delete: func(context.Context, gorp.Tx, []resource.ID) error {
				return nil
			},
			DB: db,
			Policies: impact.Policies{
				Links: links,
				KeptTables: func(table string) bool {
					return strings.HasSuffix(table, "_config")
				},
			},
		})
		return err
	}
	display := gorp.ReferenceID{Table: "display", Name: "channel"}
	keeper := gorp.ReferenceID{Table: "keeper", Name: "channel"}

	It("Should open when it maps every reference to channels", func() {
		Expect(open(map[gorp.ReferenceID]impact.Policy{
			display: impact.PolicyRetarget,
			keeper:  impact.PolicyKeep,
		})).To(Succeed())
	})

	It("Should fail on a reference to channels that it leaves out", func() {
		Expect(open(map[gorp.ReferenceID]impact.Policy{
			display: impact.PolicyRetarget,
		})).To(MatchError("[impact] - no policy for keeper.channel"))
	})

	It("Should fail on a retarget of a reference that cannot retarget", func() {
		Expect(open(map[gorp.ReferenceID]impact.Policy{
			display: impact.PolicyRetarget,
			keeper:  impact.PolicyRetarget,
		})).To(MatchError("[impact] - keeper.channel cannot retarget"))
	})

	It("Should fail on a row for a reference in a table that keeps every link", func() {
		Expect(open(map[gorp.ReferenceID]impact.Policy{
			display:                                 impact.PolicyRetarget,
			keeper:                                  impact.PolicyKeep,
			{Table: "read_config", Name: "channel"}: impact.PolicyKeep,
		})).To(MatchError(
			"[impact] - rows for read_config.channel, whose tables keep every link",
		))
	})

	It("Should retarget only the links that it maps to a retarget", func() {
		policies := impact.Policies{Links: map[gorp.ReferenceID]impact.Policy{
			display: impact.PolicyRetarget,
			keeper:  impact.PolicyKeep,
		}}
		Expect(policies.Retargets(gorp.Dependent{
			Holder:    gorp.Target{Table: "display", Key: "a"},
			Reference: "channel",
		})).To(BeTrue())
		Expect(policies.Retargets(gorp.Dependent{
			Holder:    gorp.Target{Table: "keeper", Key: "a"},
			Reference: "channel",
		})).To(BeFalse())
	})

	It("Should fail on a reference that it maps to no policy", func() {
		policies := impact.Policies{Links: map[gorp.ReferenceID]impact.Policy{
			display: impact.PolicyRetarget,
		}}
		Expect(policies.Retargets(gorp.Dependent{
			Holder:    gorp.Target{Table: "keeper", Key: "a"},
			Reference: "channel",
		})).Error().To(MatchError("[impact] - no policy for keeper.channel"))
	})
})
