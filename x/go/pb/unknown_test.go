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
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ = Describe("RejectUnknown", func() {
	unknown := func() *structpb.Value {
		v := structpb.NewStringValue("cold flow")
		v.ProtoReflect().SetUnknown(
			protowire.AppendVarint(protowire.AppendTag(nil, 99, protowire.VarintType), 1),
		)
		return v
	}

	It("Should accept a message with no unknown fields", func() {
		s := &structpb.Struct{Fields: map[string]*structpb.Value{
			"a": structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{structpb.NewStringValue("b")},
			}),
		}}
		Expect(pb.RejectUnknown(s)).To(Succeed())
	})

	DescribeTable("Should reject an unknown field at any depth", func(
		build func() proto.Message,
	) {
		Expect(pb.RejectUnknown(build())).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("google.protobuf.Value has unknown fields")),
		))
	},
		Entry("at the top level", func() proto.Message { return unknown() }),
		Entry("in a message field", func() proto.Message {
			return structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{structpb.NewStringValue("a"), unknown()},
			})
		}),
		Entry("in a map value", func() proto.Message {
			return &structpb.Struct{
				Fields: map[string]*structpb.Value{"a": unknown()},
			}
		}),
	)
})
