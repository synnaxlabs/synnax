// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package query_test

import (
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/query"
	"github.com/synnaxlabs/oracle/plugin/go/types"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("@group namespaced breaker 2", func() {
	var loader *MockFileLoader
	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/resource", `
			@go output "core/pkg/service/resource"

			Type enum {
				channel = "channel"
				group   = "group"
			}
		`)
		loader.Add("schemas/group", `
			import "schemas/resource"

			@go output "core/pkg/service/group"

			Key = uuid

			Group struct {
				key   Key {
					@key
				}
				name  string
				scope resource.Type

				@resource type "group"
			}
		`)
	})
	content := func(resp *plugin.Response, suffix string) string {
		GinkgoHelper()
		for _, f := range resp.Files {
			if strings.HasSuffix(f.Path, suffix) {
				return string(f.Content)
			}
		}
		Fail("no file " + suffix)
		return ""
	}
	generate := func(source string) (string, string) {
		GinkgoHelper()
		req := MustGenerateRequest(source, "channel", loader)
		typesResp := MustSucceed(types.New(types.DefaultOptions()).Generate(req))
		queryResp := MustSucceed(query.New(query.DefaultOptions()).Generate(req))
		return content(typesResp, "types.gen.go"), content(queryResp, "retrieve.gen.go")
	}

	It("Should not put an optional name in a Sibling as it is", func() {
		typesGo, retrieveGo := generate(`
			import "schemas/group"

			@go output "core/pkg/service/channel"

			Channel struct {
				key   uint32 {
					@key
				}
				name  string?
				group group.Key? {
					@group scope channel
					@group namespaced
				}

				@resource type "channel"
				@retrieve
			}
		`)
		pointer := regexp.MustCompile(`Name\s+\*string`)
		Expect(pointer.MatchString(typesGo)).To(BeTrue())
		Expect(strings.Contains(retrieveGo, "group.Sibling{Name: e.Name}")).To(
			BeFalse(),
			"retrieve.gen.go puts e.Name in group.Sibling.Name as it is",
		)
	})

	It("Should not put a name of a distinct string type in a Sibling as it is", func() {
		typesGo, retrieveGo := generate(`
			import "schemas/group"

			@go output "core/pkg/service/channel"

			Label string {}

			Channel struct {
				key   uint32 {
					@key
				}
				name  Label
				group group.Key? {
					@group scope channel
					@group namespaced
				}

				@resource type "channel"
				@retrieve
			}
		`)
		Expect(strings.Contains(typesGo, "type Label string")).To(BeTrue())
		distinct := regexp.MustCompile(`Name\s+Label`)
		Expect(distinct.MatchString(typesGo)).To(BeTrue())
		Expect(strings.Contains(retrieveGo, "group.Sibling{Name: e.Name}")).To(
			BeFalse(),
			"retrieve.gen.go puts e.Name in group.Sibling.Name as it is",
		)
	})

	It("Should refuse a namespaced member that no retrieve gives a sibling index",
		func() {
			req := MustGenerateRequest(`
				import "schemas/group"

				@go output "core/pkg/service/channel"

				Channel struct {
					key   uint32 {
						@key
					}
					name  string
					group group.Key? {
						@group scope channel
						@group namespaced
					}

					@resource type "channel"
				}
			`, "channel", loader)
			Expect(
				query.New(query.DefaultOptions()).Generate(req),
			).Error().
				To(MatchError(ContainSubstring(
					"channel.Channel: @group namespaced needs @retrieve on the record",
				)))
		},
	)
})
