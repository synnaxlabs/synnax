// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Update", func() {
	var r rack.Rack
	BeforeEach(func() {
		r = rack.Rack{
			Key:          1,
			Name:         "Rack",
			Integrations: []string{"ni"},
			Version:      new("1.0.0"),
		}
	})
	update := func(u rack.Update) (rack.Rack, error) {
		return rack.Reduce(r, rack.NewUpdateAction(rack.UpdatePayload{Value: u}))
	}

	It("Should set a scalar field", func() {
		next := MustSucceed(update(rack.Update{Name: new("Rack 2")}))
		Expect(next.Name).To(Equal("Rack 2"))
		Expect(next.Version).To(Equal(new("1.0.0")))
	})

	It("Should clear an optional field", func() {
		next := MustSucceed(update(rack.Update{Version: patch.Clear[string]()}))
		Expect(next.Version).To(BeNil())
		Expect(next.Name).To(Equal("Rack"))
	})

	It("Should insert into a list of values", func() {
		next := MustSucceed(update(rack.Update{
			Integrations: &patch.ValueList[string]{Insert: []string{"labjack"}},
		}))
		Expect(next.Integrations).To(Equal([]string{"ni", "labjack"}))
		Expect(r.Integrations).To(Equal([]string{"ni"}))
	})

	It("Should leave the rack alone when the update fails", func() {
		next, err := update(rack.Update{
			Name:         new("Rack 2"),
			Integrations: &patch.ValueList[string]{Insert: []string{"ni"}},
		})
		Expect(err).To(And(
			MatchError(query.ErrUniqueViolation),
			MatchError(ContainSubstring("ni already exists")),
		))
		Expect(next).To(Equal(r))
	})
})
