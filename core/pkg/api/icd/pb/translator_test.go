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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/icd"
	"github.com/synnaxlabs/synnax/pkg/api/icd/pb"
	serviceicd "github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Translator", func() {
	DescribeTable(
		"Should round trip a retrieve request",
		func(req icd.RetrieveRequest) {
			translated := MustSucceed(pb.RetrieveRequestToPB(req))
			Expect(pb.RetrieveRequestFromPB(translated)).To(Equal(req))
		},
		Entry("that selects every ICD", icd.RetrieveRequest{}),
		Entry("that filters, searches, and pages", icd.RetrieveRequest{
			Where: &serviceicd.Where{
				Key: &filter.Set[serviceicd.Key]{In: []serviceicd.Key{uuid.New()}},
			},
			SearchTerm: "rig",
			After:      &icd.Cursor{Key: uuid.New()},
			Limit:      new(uint32(10)),
		}),
	)

	It("Should round trip a retrieve response", func() {
		res := icd.RetrieveResponse{ICDs: []serviceicd.ICD{
			{Key: uuid.New(), Name: "Rig", Messages: []serviceicd.Message{}},
		}}
		translated := MustSucceed(pb.RetrieveResponseToPB(res))
		Expect(pb.RetrieveResponseFromPB(translated)).To(Equal(res))
	})

	DescribeTable("Should reject a cursor key that is not a UUID",
		func(check func()) { check() },
		Entry("in a cursor", func() {
			Expect(pb.CursorFromPB(&pb.Cursor{Key: "rig"})).Error().
				To(MatchError("invalid uuid"))
		}),
		Entry("in a retrieve request", func() {
			Expect(pb.RetrieveRequestFromPB(&pb.RetrieveRequest{
				After: &pb.Cursor{Key: "rig"},
			})).Error().To(MatchError("invalid uuid"))
		}),
	)
})
