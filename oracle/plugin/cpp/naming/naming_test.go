// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package naming_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/cpp/naming"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/x/testutil"
)

func TestNaming(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CPP Naming Suite", SuiteConfig())
}

var _ = Describe("VariantTypeName", func() {
	DescribeTable(
		"should derive the C++ variant struct name",
		func(union, variant, expected string) {
			Expect(naming.VariantTypeName(union, variant)).To(Equal(expected))
		},
		Entry("plain union takes the variant first", "Scale", "linear", "LinearScale"),
		Entry(
			"acronym union factors the shared prefix",
			"AIChannel",
			"ai_voltage",
			"AIVoltageChannel",
		),
		Entry(
			"variant not repeating the acronym prefixes the whole union name",
			"AIChannel",
			"voltage",
			"VoltageAIChannel",
		),
		Entry("reserved-word variant value", "Scale", "map", "MapScale"),
	)
})

var _ = Describe("QualifiedVariantTypeName", func() {
	DescribeTable(
		"should derive the variant name under the union's namespace",
		func(union, variant, expected string) {
			name := naming.QualifiedVariantTypeName(union, variant)
			Expect(name).To(Equal(expected))
		},
		Entry(
			"namespaced union",
			"::ni::AIChannel",
			"ai_voltage",
			"::ni::AIVoltageChannel",
		),
		Entry("unqualified union", "Scale", "linear", "LinearScale"),
	)
})

var _ = Describe("Namespace", func() {
	DescribeTable(
		"should derive the namespace from the output path",
		func(outputPath, expected string) {
			Expect(naming.Namespace(outputPath)).To(Equal(expected))
		},
		Entry("x/cpp path", "x/cpp/telem", "x::telem"),
		Entry("x/cpp root", "x/cpp", "x"),
		Entry("client/cpp path", "client/cpp/channel", "synnax::channel"),
		Entry("nested client/cpp path", "client/cpp/task/common",
			"synnax::task::common"),
		Entry("arc/cpp path", "arc/cpp/runtime", "arc::runtime"),
		Entry("driver path", "driver/ni", "driver::ni"),
		Entry("driver root", "driver", "driver"),
		Entry("unknown path keeps the last segment", "foo/bar", "synnax::bar"),
		Entry("single unknown segment", "core", "synnax::core"),
	)
})

var _ = Describe("PBNamespace", func() {
	DescribeTable(
		"should derive the C++ namespace from the pb output path and namespace",
		func(pbOutputPath, namespace, expected string) {
			Expect(naming.PBNamespace(pbOutputPath, namespace)).To(Equal(expected))
		},
		Entry("empty path", "", "channel", "pb"),
		Entry(
			"core layer path",
			"core/pkg/distribution/channel/pb",
			"channel",
			"::distribution::channel::pb",
		),
		Entry(
			"nested core path keeps the namespace",
			"core/pkg/service/legacy/device/pb",
			"device",
			"::service::device::pb",
		),
		Entry(
			"namespace differs from the output directory",
			"core/pkg/service/legacy/pb",
			"device",
			"::service::device::pb",
		),
		Entry("x/go path", "x/go/telem/pb", "telem", "::x::telem::pb"),
		Entry(
			"two-segment path",
			"freighter/pb",
			"freighter",
			"::freighter::freighter::pb",
		),
		Entry(
			"leading slash falls back to synnax",
			"/foo/pb",
			"foo",
			"::synnax::foo::pb",
		),
	)
})

var _ = Describe("VariantPBNamespace", func() {
	pbType := func(name, namespace, goOutput string) resolution.Type {
		return resolution.Type{
			Name:      name,
			Namespace: namespace,
			Domains: map[string]resolution.Domain{
				"go": {
					Name: "go",
					Expressions: resolution.Expressions{{
						Name:   "output",
						Values: []resolution.ExpressionValue{{StringValue: goOutput}},
					}},
				},
				"pb": {Name: "pb"},
			},
		}
	}
	union := pbType("Node", "graph", "core/pkg/service/graph")
	payload := pbType("Valve", "base", "core/pkg/service/base")

	DescribeTable("should derive the message owner from the variant",
		func(inline bool, expectedNamespace, expectedHeader string) {
			namespace, header := naming.VariantPBNamespace(
				union,
				resolution.UnionVariant{Name: "valve", Inline: inline},
				payload,
			)
			Expect(namespace).To(Equal(expectedNamespace))
			Expect(header).To(Equal(expectedHeader))
		},
		Entry(
			"an inline variant belongs to its union",
			true,
			"::service::graph::pb",
			"core/pkg/service/graph/pb/graph.pb.h",
		),
		Entry(
			"a referenced variant belongs to its payload",
			false,
			"::service::base::pb",
			"core/pkg/service/base/pb/base.pb.h",
		),
	)
})

var _ = Describe("PBName", func() {
	It("should return the @pb name override", func() {
		typ := resolution.Type{
			Name: "Status",
			Domains: map[string]resolution.Domain{
				"pb": {
					Name: "pb",
					Expressions: resolution.Expressions{{
						Name:   "name",
						Values: []resolution.ExpressionValue{{StringValue: "PBStatus"}},
					}},
				},
			},
		}
		Expect(naming.PBName(typ)).To(Equal("PBStatus"))
	})

	It("should return the declared name without a pb domain", func() {
		Expect(naming.PBName(resolution.Type{Name: "Status"})).To(Equal("Status"))
	})

	It("should return the declared name when the pb domain has no name", func() {
		typ := resolution.Type{
			Name: "Status",
			Domains: map[string]resolution.Domain{
				"pb": {
					Name:        "pb",
					Expressions: resolution.Expressions{{Name: "output"}},
				},
			},
		}
		Expect(naming.PBName(typ)).To(Equal("Status"))
	})
})

var _ = Describe("ScreamingSnake", func() {
	DescribeTable(
		"should render enum constant names",
		func(input, expected string) {
			Expect(naming.ScreamingSnake(input)).To(Equal(expected))
		},
		Entry("camelCase", "linearScale", "LINEAR_SCALE"),
		Entry("PascalCase", "LinearScale", "LINEAR_SCALE"),
		Entry("single word", "linear", "LINEAR"),
		Entry("snake_case passes through", "linear_scale", "LINEAR_SCALE"),
	)
})

var _ = ShouldNotLeakGoroutinesPerSpec()
