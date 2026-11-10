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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ServiceConfig", func() {
	Describe("Validate", func() {
		It("Should succeed when all required fields are set", func() {
			cfg := log.ServiceConfig{
				DB:   db,
				ImEx: imex.NewService(),
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("Should fail when DB is nil", func() {
			cfg := log.ServiceConfig{
				ImEx: imex.NewService(),
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("db")))
		})

		It("Should fail when ImEx is nil", func() {
			cfg := log.ServiceConfig{
				DB: db,
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("imex")))
		})
	})

	Describe("Override", func() {
		It("Should prefer the other config's fields when set", func() {
			initial := log.ServiceConfig{}
			other := log.ServiceConfig{
				DB:   db,
				ImEx: imex.NewService(),
			}
			merged := initial.Override(other)
			Expect(merged.DB).To(BeIdenticalTo(other.DB))
			Expect(merged.ImEx).To(BeIdenticalTo(other.ImEx))
		})

		It("Should preserve the receiver's fields when other's are nil", func() {
			initial := log.ServiceConfig{
				DB:   db,
				ImEx: imex.NewService(),
			}
			merged := initial.Override(log.ServiceConfig{})
			Expect(merged.DB).To(BeIdenticalTo(initial.DB))
			Expect(merged.ImEx).To(BeIdenticalTo(initial.ImEx))
		})
	})
})

var _ = Describe("OpenService", func() {
	It(
		"Should fail when ServiceConfig is missing required fields",
		func(ctx SpecContext) {
			Expect(log.OpenService(ctx, log.ServiceConfig{DB: db})).Error().To(
				MatchError(ContainSubstring("imex")),
			)
		},
	)

	It(
		"Should register itself with the configured ImEx registry on open",
		func(ctx SpecContext) {
			l := defaults.Fill(log.Log{Project: &proj.Key, Name: "auto-registered"})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &l)
			})).To(Succeed())
			env := MustSucceed(imexSvc.Export(ctx, db, l.ResourceID()))
			Expect(env.Type).To(Equal("log"))
		},
	)

	It("Should wire up signals when a provider is configured", func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: node.DB,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB: node.DB,
		}))
		channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statusSvc,
		}))
		sigs := MustSucceed(signals.New(signals.Config{
			DB:      node.DB,
			Channel: channelSvc,
			Framer:  node.Framer,
		}))
		MustOpen(log.OpenService(ctx, log.ServiceConfig{
			DB:      node.DB,
			ImEx:    imex.NewService(),
			Signals: sigs,
		}))
		for _, name := range []string{"sy_log_set", "sy_log_delete"} {
			var ch channel.Channel
			Expect(channelSvc.NewRetrieve().Where(channel.MatchNames(name)).
				Entry(&ch).Exec(ctx, node.DB)).To(Succeed())
			Expect(ch.Virtual).To(BeTrue())
			Expect(ch.Internal).To(BeTrue())
		}
	})
})
