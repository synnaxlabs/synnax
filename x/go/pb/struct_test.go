// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/pb"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/types/known/structpb"
)

type action struct {
	Type string    `json:"type"`
	Set  *setValue `json:"set,omitzero"`
}

type setValue struct {
	Path  []string `json:"path"`
	Value any      `json:"value"`
}

var _ = Describe("Structs", func() {
	acts := []action{
		{Type: "set", Set: &setValue{Path: []string{"name"}, Value: "Rack"}},
		{Type: "set", Set: &setValue{Path: []string{"port"}, Value: 2.0}},
	}

	Describe("ToStructs", func() {
		It("Should hold the JSON form of each value", func() {
			structs := MustSucceed(pb.ToStructs(acts))
			Expect(structs).To(HaveLen(2))
			Expect(structs[0].AsMap()).To(Equal(map[string]any{
				"type": "set",
				"set":  map[string]any{"path": []any{"name"}, "value": "Rack"},
			}))
		})

		It("Should return no structs for no values", func() {
			Expect(pb.ToStructs[action](nil)).To(BeEmpty())
		})

		It("Should refuse a value that is not a JSON object", func() {
			Expect(pb.ToStructs([]string{"set"})).Error().
				To(MatchError(ContainSubstring(`unexpected token "set"`)))
		})
	})

	Describe("FromStructs", func() {
		It("Should decode each struct into a value", func() {
			Expect(pb.FromStructs[action](MustSucceed(pb.ToStructs(acts)))).
				To(Equal(acts))
		})

		It("Should refuse a struct that does not decode", func() {
			bad := MustSucceed(structpb.NewStruct(map[string]any{"type": 1.0}))
			Expect(pb.FromStructs[action]([]*structpb.Struct{bad})).Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("0: ")),
				))
		})
	})
})
