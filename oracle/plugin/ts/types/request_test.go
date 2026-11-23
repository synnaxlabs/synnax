// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/ts/types"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Request", func() {
	var loader *MockFileLoader

	BeforeEach(func() {
		loader = NewMockFileLoader()
		AddFilterSchema(loader)
	})

	const thing = `
		import "schemas/x/filter"

		@ts output "client/ts/src/thing"

		Label string
		Span struct {
			start int64
		}
		Thing struct {
			key    uuid     { @key @ts type "string" }
			serial uint64   { @ts type "number" }
			counts int64[]  { @ts type "number" }
			tag    uint32   { @ts type "Label" }
			code   uint32   { @ts type "thing.Label" }
			count  int64
			name   string   { @ts type "string" }
		}
		Where filter Thing {
			key
			serial
			counts
			tag
			code
			count
			name
			overlaps Span
		}
	`

	// lineArg returns the type argument of the operator of the line name.
	lineArg := func(source, name string) resolution.TypeRef {
		GinkgoHelper()
		prepared := MustSucceed(
			types.Request(MustGenerateRequest(source, "thing", loader)),
		)
		where := prepared.Resolutions.MustGet("thing.Where")
		field := MustBeOk(resolution.MustForm[resolution.StructForm](where).Field(name))
		Expect(field.Type.TypeArgs).To(HaveLen(1))
		return field.Type.TypeArgs[0]
	}

	DescribeTable(
		"Should type the operator of a bare line with the @ts type of its field",
		func(line, arg string) {
			Expect(lineArg(thing, line).Name).To(Equal(arg))
		},
		Entry("a number on an ordered line", "serial", "float64"),
		Entry("a string on a set line", "key", "string"),
		Entry("a number on the elements of a contains line", "counts", "float64"),
		Entry("a type of the same namespace", "tag", "thing.Label"),
		Entry("a qualified type", "code", "thing.Label"),
		Entry("no override", "count", "int64"),
	)

	It("Should leave a typed line and an operator with no type argument", func() {
		prepared := MustSucceed(
			types.Request(MustGenerateRequest(thing, "thing", loader)),
		)
		form := resolution.MustForm[resolution.StructForm](
			prepared.Resolutions.MustGet("thing.Where"),
		)
		Expect(MustBeOk(form.Field("name")).Type.TypeArgs).To(BeEmpty())
		Expect(MustBeOk(form.Field("overlaps")).Type.Name).To(Equal("thing.Span"))
	})

	It("Should read the @ts type of the record that the response holds", func() {
		source := `
			import "schemas/x/filter"

			@ts output "client/ts/src/thing"

			Thing struct {
				key    uuid { @key }
				serial uint64
			}
			Wire struct {
				key    uuid
				serial uint64 { @ts type "number" }
			}
			Where filter Thing {
				serial

				@response type Wire
			}
		`
		Expect(lineArg(source, "serial").Name).To(Equal("float64"))
	})

	It("Should reject a @ts type that names no type", func() {
		source := strings.Replace(thing, `"thing.Label"`, `"nothing"`, 1)
		Expect(types.Request(MustGenerateRequest(source, "thing", loader))).
			Error().To(MatchError(
			"filter thing.Where: line code reads a field whose @ts type nothing is " +
				"not a type",
		))
	})
})
