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
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	. "github.com/synnaxlabs/x/testutil"
)

// actionVector is an action envelope that the Core and every client decode alike.
type actionVector struct {
	Name   string         `json:"name"`
	Action jsontext.Value `json:"action"`
	// Value is the value that a set or insert payload holds after decoding.
	Value jsontext.Value `json:"value"`
}

var _ = Describe("Action vectors", func() {
	It("Should decode every action that the clients share", func() {
		var vectors map[string][]actionVector
		Expect(json.Unmarshal(
			MustSucceed(os.ReadFile("../../../testdata/actions/vectors.json")),
			&vectors,
		)).To(Succeed())
		Expect(vectors["rack"]).ToNot(BeEmpty())
		for _, v := range vectors["rack"] {
			var a rack.Action
			Expect(json.Unmarshal(v.Action, &a)).To(Succeed(), v.Name)
			var envelope struct {
				Type rack.ActionType `json:"type"`
			}
			Expect(json.Unmarshal(
				v.Action, &envelope, json.RejectUnknownMembers(false),
			)).To(Succeed())
			Expect(a.Type()).To(Equal(envelope.Type), v.Name)
			if len(v.Value) == 0 {
				continue
			}
			var got any
			switch p := a.Variant.(type) {
			case rack.SetPayload:
				got = p.Value
			case rack.InsertPayload:
				got = p.Value
			default:
				Fail(v.Name + ": only a set or insert holds a value")
			}
			var want any
			Expect(json.Unmarshal(v.Value, &want)).To(Succeed())
			if want == nil {
				Expect(got).To(BeNil(), v.Name)
			} else {
				Expect(got).To(Equal(want), v.Name)
			}
		}
	})
})
