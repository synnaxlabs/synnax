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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ServiceConfig", func() {
	Describe("Validate", func() {
		It("Should succeed when all required fields are set", func() {
			cfg := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   &search.Index{},
				ImEx:     imex.NewService(otg),
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("Should fail when DB is nil", func() {
			cfg := log.ServiceConfig{
				Ontology: otg,
				Search:   &search.Index{},
				ImEx:     imex.NewService(otg),
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("db")))
		})

		It("Should fail when Ontology is nil", func() {
			cfg := log.ServiceConfig{
				DB:     db,
				Search: &search.Index{},
				ImEx:   imex.NewService(otg),
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("ontology")))
		})

		It("Should fail when Search is nil", func() {
			cfg := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				ImEx:     imex.NewService(otg),
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("search")))
		})

		It("Should fail when ImEx is nil", func() {
			cfg := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   &search.Index{},
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("imex")))
		})
	})

	Describe("Override", func() {
		It("Should prefer the other config's fields when set", func() {
			initial := log.ServiceConfig{}
			other := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   &search.Index{},
				ImEx:     imex.NewService(otg),
			}
			merged := initial.Override(other)
			Expect(merged.DB).To(BeIdenticalTo(other.DB))
			Expect(merged.Ontology).To(BeIdenticalTo(other.Ontology))
			Expect(merged.Search).To(BeIdenticalTo(other.Search))
			Expect(merged.ImEx).To(BeIdenticalTo(other.ImEx))
		})

		It("Should preserve the receiver's fields when other's are nil", func() {
			initial := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   &search.Index{},
				ImEx:     imex.NewService(otg),
			}
			merged := initial.Override(log.ServiceConfig{})
			Expect(merged.DB).To(BeIdenticalTo(initial.DB))
			Expect(merged.Ontology).To(BeIdenticalTo(initial.Ontology))
			Expect(merged.Search).To(BeIdenticalTo(initial.Search))
			Expect(merged.ImEx).To(BeIdenticalTo(initial.ImEx))
		})
	})
})

var _ = Describe("OpenService", func() {
	It("Should fail when ServiceConfig is missing required fields", func(ctx SpecContext) {
		Expect(log.OpenService(ctx, log.ServiceConfig{DB: db})).Error().To(
			MatchError(ContainSubstring("ontology")),
		)
	})

	It("Should register itself with the configured ImEx registry on open", func(ctx SpecContext) {
		l := log.Log{Name: "auto-registered"}
		Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &l)).To(Succeed())
		env := MustSucceed(imexSvc.Export(ctx, log.OntologyID(l.Key)))
		Expect(env.Type).To(Equal("log"))
	})

	It("Should wire up signals when a provider is configured", func(ctx SpecContext) {
		node := mock.NewNode(ctx)
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       node.DB,
			Ontology: node.Ontology,
			Group:    node.Group,
			Search:   node.Search,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       node.DB,
			Ontology: node.Ontology,
			Group:    node.Group,
			Label:    labelSvc,
			Search:   node.Search,
		}))
		channelSvc := MustSucceed(channel.NewService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			Status:  statusSvc,
		}))
		framerSvc := MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
			Framer:  node.Framer,
			Channel: channelSvc,
			Status:  statusSvc,
		}))
		sigs := MustSucceed(signals.New(signals.Config{
			Channel: channelSvc,
			Framer:  framerSvc,
		}))
		MustOpen(log.OpenService(ctx, log.ServiceConfig{
			DB:       node.DB,
			Ontology: node.Ontology,
			Search:   node.Search,
			ImEx:     imex.NewService(otg),
			Signals:  sigs,
		}))
		for _, name := range []string{"sy_log_set", "sy_log_delete"} {
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().Where(channel.MatchNames(name)).
				Entry(&ch).Exec(ctx, nil)).To(Succeed())
			Expect(ch.Virtual).To(BeTrue())
			Expect(ch.Internal).To(BeTrue())
		}
	})
})
