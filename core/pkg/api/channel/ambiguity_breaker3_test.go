// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ambiguity breaker 3", func() {
	var (
		u             user.User
		shown, hidden group.Group
		first, second string
	)
	BeforeEach(func(ctx SpecContext) {
		u = MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		shown = group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		hidden = group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := groupSvc.NewWriter(tx)
			for _, g := range []*group.Group{&shown, &hidden} {
				if err := w.Create(ctx, g); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		first, second = UniqueChannelName(), UniqueChannelName()
		stored := func(n string, key channel.LocalKey, g *group.Key) channel.Channel {
			return channel.Channel{
				Name:        n,
				Leaseholder: node.KeyFree,
				LocalKey:    key,
				DataType:    telem.Float32T,
				Virtual:     true,
				Group:       g,
			}
		}
		holders := []channel.Channel{
			stored(first, 1_000_141, &shown.Key),
			stored(first, 1_000_142, &hidden.Key),
			stored(second, 1_000_143, &shown.Key),
			stored(second, 1_000_144, &hidden.Key),
		}
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
			Set(ctx, holders...)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
				Delete(ctx, channel.KeysFromChannels(holders)...)).To(Succeed())
		})
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			resource.ID{Type: resource.TypeChannel},
			shown.ResourceID(),
		)
	})
	createCalc := func(ctx SpecContext, expression string) error {
		_, err := createChannels(AuthedCtx(ctx, u), apichannel.CreateRequest{
			Channels: []apichannel.Channel{{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: expression,
			}},
		})
		return err
	}

	It("Should report an ambiguous name of a calculated channel as a diagnostic, as "+
		"it does an unknown name", func(ctx SpecContext) {
		err := createCalc(ctx, "return "+first+" + breaker_unknown_input")
		diags, isDiagnostics := errors.AsType[*diagnostics.Diagnostics](err)
		Expect(isDiagnostics).To(BeTrue(), "expected diagnostics, got %v", err)
		// The analysis reports the unknown name. The client's error drops it.
		Expect(diags.Error()).To(ContainSubstring("breaker_unknown_input"))
		Expect(err).To(MatchError(ContainSubstring("breaker_unknown_input")))
	})

	It("Should quote the shown path of each ambiguous name of a calculated channel",
		func(ctx SpecContext) {
			err := createCalc(ctx, "return "+first+" + "+second)
			diags, isDiagnostics := errors.AsType[*diagnostics.Diagnostics](err)
			Expect(isDiagnostics).To(BeTrue(), "expected diagnostics, got %v", err)
			// The analysis reports both names. The client's error names one.
			Expect(diags.Error()).To(SatisfyAll(
				ContainSubstring("named '"+first+"'"),
				ContainSubstring("named '"+second+"'"),
			))
			for _, n := range []string{first, second} {
				Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf(
					"2 channels are named '%s': '%s.%s', and 1 more you cannot see",
					n,
					shown.Name,
					n,
				))))
			}
		})
})
