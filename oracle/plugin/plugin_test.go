// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package plugin_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("Request", func() {
	Describe("OmitFields", func() {
		tsOmit := map[string]resolution.Domain{
			"ts": {Expressions: []resolution.Expression{{Name: "omit"}}},
		}
		newRequest := func() *plugin.Request {
			return &plugin.Request{
				RepoRoot: "/repo",
				Resolutions: &resolution.Table{Types: []resolution.Type{{
					QualifiedName: "auth.Session",
					Form: resolution.StructForm{Fields: []resolution.Field{
						{Name: "key"}, {Name: "hash", Domains: tsOmit},
					}},
				}}},
			}
		}
		fields := func(r *plugin.Request) []resolution.Field {
			return r.Resolutions.MustGet("auth.Session").Form.(resolution.StructForm).Fields
		}

		It("Should drop the fields omitted in the languages", func() {
			stripped := newRequest().OmitFields("ts")
			Expect(fields(stripped)).To(HaveLen(1))
			Expect(fields(stripped)[0].Name).To(Equal("key"))
			Expect(stripped.RepoRoot).To(Equal("/repo"))
		})

		It("Should keep fields omitted only in other languages", func() {
			Expect(fields(newRequest().OmitFields("py"))).To(HaveLen(2))
		})

		It("Should leave the original request unchanged", func() {
			req := newRequest()
			req.OmitFields("ts")
			Expect(fields(req)).To(HaveLen(2))
		})
	})
})
