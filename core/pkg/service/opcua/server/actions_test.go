// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Actions", func() {
	var base server.Server
	BeforeEach(func() {
		base = server.Server{
			Key:        uuid.New(),
			Rack:       1,
			Name:       "Boiler PLC",
			Connection: newConnection("opc.tcp://boiler:4840"),
			Reads: []server.Read{
				{Node: "nsu=urn:plc;s=Temp", Index: 1, Channel: 2},
			},
		}
	})
	set := func(path []string, value any) server.Action {
		return server.NewSetAction(server.SetPayload{Path: path, Value: value})
	}

	It("Should set the name and keep the other fields", func() {
		res := MustSucceed(server.Reduce(base, set([]string{"name"}, "Mixer PLC")))
		Expect(res.Name).To(Equal("Mixer PLC"))
		Expect(res.Connection).To(Equal(base.Connection))
		Expect(res.Reads).To(Equal(base.Reads))
	})

	It("Should set one connection field and keep the others", func() {
		res := MustSucceed(server.Reduce(
			base,
			set([]string{"connection", "session_timeout"}, 30*telem.Minute),
		))
		Expect(res.Connection.SessionTimeout).To(Equal(30 * telem.Minute))
		Expect(res.Connection.Endpoint).To(Equal(base.Connection.Endpoint))
	})

	It("Should replace the writes and keep the reads", func() {
		res := MustSucceed(server.Reduce(base, set(
			[]string{"writes"},
			[]map[string]any{{"node": "nsu=urn:plc;s=Valve", "channel": 4}},
		)))
		Expect(res.Writes).To(Equal([]server.Write{
			{Node: "nsu=urn:plc;s=Valve", Channel: 4},
		}))
		Expect(res.Reads).To(Equal(base.Reads))
	})

	It("Should change the login by its type", func() {
		res := MustSucceed(server.Reduce(base, set(
			[]string{"connection", "auth"},
			map[string]any{"type": "username", "username": "op"},
		)))
		Expect(res.Connection.Auth.Variant).
			To(Equal(server.UsernameAuth{Username: "op"}))
	})

	DescribeTable("Should reject a path that names no editable field",
		func(path []string) {
			Expect(server.Reduce(base, set(path, "x"))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("the path names no editable field")),
			))
		},
		Entry("the key", []string{"key"}),
		Entry("the status", []string{"status"}),
	)
})
