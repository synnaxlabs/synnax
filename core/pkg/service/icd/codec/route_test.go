// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec_test

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	xjson "github.com/synnaxlabs/x/json"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// routeTable is testdata/routes.json, which the Console's route spec also reads.
type routeTable struct {
	Groups []routeGroup `json:"groups"`
}

type routeGroup struct {
	Description string           `json:"description"`
	Messages    []fixtureMessage `json:"messages"`
	Cases       []routeCase      `json:"cases"`
}

// routeCase is a document, the pointers of the fields that inference gives it, and the
// message that it belongs to. Unmatched is true when it belongs to no message. Matches
// names the two messages whose present rules it meets, when the Core refuses it.
type routeCase struct {
	Document  json.RawMessage `json:"document"`
	Fields    []string        `json:"fields"`
	Message   string          `json:"message"`
	Unmatched bool            `json:"unmatched"`
	Matches   []string        `json:"matches"`
}

var _ = Describe("Routes", func() {
	var t routeTable
	Expect(json.Unmarshal(
		MustSucceed(os.ReadFile("../testdata/routes.json")),
		&t,
	)).To(Succeed())
	groups := lo.Map(t.Groups, func(g routeGroup, _ int) TableEntry {
		return Entry(g.Description, g)
	})

	DescribeTable("Should give each document the message of the table",
		func(g routeGroup) {
			messages := lo.Map(
				g.Messages,
				func(m fixtureMessage, _ int) codec.JSONMessage { return m.jsonMessage() },
			)
			dec := MustSucceed(codec.NewJSON(messages))
			for k, c := range g.Cases {
				By("case " + strconv.Itoa(k))
				if len(c.Matches) > 0 {
					_, _, _, err := dec.Decode(c.Document, newSamples(messages))
					Expect(err).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring(
							"document matches messages %s",
							strings.Join(c.Matches, " and "),
						)),
					))
				} else {
					m, ok := decode(dec, c.Document, newSamples(messages))
					Expect(ok).To(Equal(!c.Unmatched))
					if ok {
						Expect(messages[m].Name).To(Equal(c.Message))
					}
				}
				shapes, _ := xjson.Infer(
					[]xjson.Document{{Source: "s", Data: c.Document}},
				)
				Expect(shapes).To(HaveLen(1))
				pointers := lo.Map(
					shapes[0].Fields,
					func(f xjson.InferredField, _ int) string { return f.Pointer.String() },
				)
				Expect(pointers).To(ConsistOf(c.Fields))
			}
		},
		groups,
	)
})
