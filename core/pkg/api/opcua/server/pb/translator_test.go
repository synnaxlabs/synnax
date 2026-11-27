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
	apiserver "github.com/synnaxlabs/synnax/pkg/api/opcua/server"
	"github.com/synnaxlabs/synnax/pkg/api/opcua/server/pb"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Translator", func() {
	DescribeTable("Should round trip a retrieve request",
		func(req apiserver.RetrieveRequest) {
			translated := MustSucceed(pb.RetrieveRequestToPB(req))
			Expect(pb.RetrieveRequestFromPB(translated)).To(Equal(req))
		},
		Entry("with no options", apiserver.RetrieveRequest{}),
		Entry("with every option", func() apiserver.RetrieveRequest {
			key := uuid.New()
			return apiserver.RetrieveRequest{
				Where:         &server.Where{Key: &filter.Set[server.Key]{Eq: &key}},
				SearchTerm:    "PLC",
				After:         &apiserver.Cursor{Key: uuid.New()},
				Limit:         new(uint32(10)),
				IncludeStatus: true,
			}
		}()),
	)

	It("Should round trip a retrieve response", func() {
		res := apiserver.RetrieveResponse{Servers: []server.Server{{
			Key:  uuid.New(),
			Rack: rack.NewKey(1, 2),
			Name: "PLC",
			Connection: server.Connection{
				Endpoint:              "opc.tcp://plc:4840",
				Auth:                  server.Auth{Variant: server.AnonymousAuth{}},
				Security:              server.Security{Variant: server.NoneSecurity{}},
				ClientTimeout:         5 * telem.Second,
				SecureChannelLifetime: 10 * telem.Minute,
				SessionTimeout:        20 * telem.Minute,
			},
			Reads:  []server.Read{},
			Writes: []server.Write{},
		}}}
		translated := MustSucceed(pb.RetrieveResponseToPB(res))
		Expect(pb.RetrieveResponseFromPB(translated)).To(Equal(res))
	})

	It("Should reject a cursor key that is not a UUID", func() {
		translated := MustSucceed(pb.CursorToPB(apiserver.Cursor{Key: uuid.New()}))
		translated.Key = "not-a-uuid"
		Expect(pb.CursorFromPB(translated)).Error().
			To(MatchError(ContainSubstring("invalid uuid")))
	})
})
