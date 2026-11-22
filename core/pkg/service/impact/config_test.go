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
)

// noMoves returns moves that do nothing, one for each type that can sit in a group.
func noMoves() map[resource.Type]impact.Move {
	moves := make(map[resource.Type]impact.Move)
	for _, t := range group.MemberTypes() {
		moves[t] = func(context.Context, gorp.Tx, []resource.ID, *group.Key) error {
			return nil
		}
	}
	return moves
}

var _ = Describe("ServiceConfig", func() {
	full := func() impact.ServiceConfig {
		return impact.ServiceConfig{
			Channel:  &channel.Service{},
			Group:    &group.Service{},
			RecordOf: func(gorp.Target) resource.ID { return resource.ID{} },
			Delete:   func(context.Context, gorp.Tx, []resource.ID) error { return nil },
			Moves:    noMoves(),
			DB:       &gorp.DB{},
			Policies: impact.Policies{Links: make(map[gorp.ReferenceID]impact.Policy)},
		}
	}

	It("Should succeed when every service is set", func() {
		Expect(full().Validate()).To(Succeed())
	})

	DescribeTable(
		"Should fail when a service is missing",
		func(unset func(*impact.ServiceConfig), msg string) {
			cfg := full()
			unset(&cfg)
			Expect(cfg.Validate()).To(MatchError(msg))
		},
		Entry(
			"channel",
			func(c *impact.ServiceConfig) { c.Channel = nil },
			"channel: must be non-nil",
		),
		Entry(
			"group",
			func(c *impact.ServiceConfig) { c.Group = nil },
			"group: must be non-nil",
		),
		Entry(
			"record_of",
			func(c *impact.ServiceConfig) { c.RecordOf = nil },
			"record_of: must be non-nil",
		),
		Entry(
			"delete",
			func(c *impact.ServiceConfig) { c.Delete = nil },
			"delete: must be non-nil",
		),
		Entry(
			"moves",
			func(c *impact.ServiceConfig) { c.Moves = nil },
			"moves: must be non-nil",
		),
		Entry(
			"the move of one type that can sit in a group",
			func(c *impact.ServiceConfig) { delete(c.Moves, resource.TypeChannel) },
			"moves: has no move for a channel",
		),
		Entry(
			"db",
			func(c *impact.ServiceConfig) { c.DB = nil },
			"db: must be non-nil",
		),
		Entry(
			"policies",
			func(c *impact.ServiceConfig) { c.Policies = impact.Policies{} },
			"policies.links: must be non-nil",
		),
	)

	It("Should take the other config's group when set", func() {
		cfg := full()
		Expect(impact.ServiceConfig{}.Override(cfg).Group).To(BeIdenticalTo(cfg.Group))
	})
})
