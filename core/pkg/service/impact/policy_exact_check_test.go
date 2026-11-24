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

var _ = Describe("Policies exact check", func() {
	It("Should fail on a row for a reference that the database does not hold", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
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
		// ServiceConfig.Policies: OpenService fails unless it maps every such reference
		// of DB exactly.
		Expect(impact.OpenService(impact.ServiceConfig{
			Channel:  &channel.Service{},
			Group:    &group.Service{},
			RecordOf: func(gorp.Target) resource.ID { return resource.ID{} },
			Delete: func(context.Context, gorp.Tx, []resource.ID) error {
				return nil
			},
			Moves: noMoves(),
			DB:    db,
			Policies: impact.Policies{Links: map[gorp.ReferenceID]impact.Policy{
				{Table: "display", Name: "channel"}: impact.PolicyRetarget,
				{Table: "display", Name: "gone"}:    impact.PolicyRetarget,
			}},
		})).Error().To(MatchError(ContainSubstring("display.gone")))
	})
})
