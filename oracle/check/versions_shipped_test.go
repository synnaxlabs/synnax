// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package check_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/check"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("VersionsGate shipped versions", func() {
	const (
		v0Path    = "schemas/synnax/versions/channel/v0.oracle"
		v1Path    = "schemas/synnax/versions/channel/v1.oracle"
		shippedV0 = `
Status struct {
    message string
}

Channel struct {
    key    uuid @key
    name   string = "a"
    status Status? {
        @go marshal omit
    }

    @go marshal
}
`
	)
	var root, releaseRoot string

	write := func(dir, rel, content string) {
		GinkgoHelper()
		full := filepath.Join(dir, rel)
		Expect(os.MkdirAll(filepath.Dir(full), 0o755)).To(Succeed())
		Expect(os.WriteFile(full, []byte(content), 0o644)).To(Succeed())
	}

	messages := func() []string {
		GinkgoHelper()
		chains := MustSucceed(versions.Discover(root))
		report := check.NewVersionsGate().Run(
			GinkgoT().Context(),
			&pipeline.Result{
				Resolutions: resolution.NewTable(),
				Chains:      chains,
				Versions: versions.NewResolver(
					chains, analyzer.NewStandardFileLoader(root),
				),
			},
			check.Env{
				RepoRoot: root,
				Shipped:  &check.Release{Tag: "core/v0.60.1", Root: releaseRoot},
			},
		)
		msgs := make([]string, len(report.Findings))
		for i, f := range report.Findings {
			msgs[i] = f.Message
		}
		return msgs
	}

	shapeChange := func(rel, reason string) string {
		return "schemas/synnax/channel v0 changes the stored shape of Channel (" +
			rel + ": " + reason + "), which core/v0.60.1 shipped"
	}

	BeforeEach(func() {
		root, releaseRoot = GinkgoT().TempDir(), GinkgoT().TempDir()
		write(releaseRoot, v0Path, shippedV0)
	})

	Describe("Allowed changes", func() {
		It("Should pass a shipped version file that only gains docs and rules", func() {
			write(root, v0Path, `
Status struct {
    message string
}

Channel struct {
    key    uuid @key
    name   string = "a" {
        @doc value      "is the channel name."
        @validate max_length 10
    }
    status Status? {
        @go marshal omit
    }

    @doc value "is a channel."
    @go marshal
}
`)
			Expect(messages()).To(BeEmpty())
		})

		It("Should pass a codec marker that changes", func() {
			write(root, v0Path, `
Status struct {
    message string
}

Channel struct {
    key    uuid @key
    name   string = "a"
    status Status? {
        @go marshal omit
    }

    @go marshal hand
}
`)
			Expect(messages()).To(BeEmpty())
		})

		It("Should pass a stored type replaced by an alias of the same shape", func() {
			write(releaseRoot, v0Path, `
ID struct {
    type string
    key  string
}

Channel struct {
    owner ID

    @go marshal
}
`)
			write(root, v0Path, `
Ref struct {
    type string
    key  string
}

ID = Ref

Channel struct {
    owner ID

    @go marshal
}
`)
			Expect(messages()).To(BeEmpty())
		})

		It("Should pass a stored root replaced by an alias of the same shape", func() {
			write(releaseRoot, v0Path, `
Channel struct {
    name string

    @go marshal
}
`)
			write(root, v0Path, `
Ref struct {
    name string

    @go marshal
}

Channel = Ref
`)
			Expect(messages()).To(BeEmpty())
		})

		DescribeTable("Should pass a generic instance with the same filled-in shape",
			func(shipped, current string) {
				write(releaseRoot, v0Path, shipped)
				write(root, v0Path, current)
				Expect(messages()).To(BeEmpty())
			},
			Entry("a struct that becomes an alias of a generic instance", `
Inner struct {
    value uint8
}

Channel struct {
    payload Inner

    @go marshal
}
`, `
Real struct<T> {
    value T
}

Inner = Real<uint8>

Channel struct {
    payload Inner

    @go marshal
}
`),
			Entry("a stored root that becomes an alias of a generic instance", `
Channel struct {
    value uint8

    @go marshal
}
`, `
Real struct<T> {
    value T

    @go marshal
}

Channel = Real<uint8>
`),
			Entry("a nested generic instance that becomes a struct", `
Wrap struct<T> {
    item T
}

Box struct<T> {
    value T
}

Channel struct {
    payload Wrap<Box<uint8>>

    @go marshal
}
`, `
Wrap struct<T> {
    item T
}

Item struct {
    value uint8
}

Channel struct {
    payload Wrap<Item>

    @go marshal
}
`),
			Entry("an omitted type argument that becomes its default", `
Bounds struct<T = float64> {
    lower T
}

Channel struct {
    bounds Bounds

    @go marshal
}
`, `
Bounds struct<T> {
    lower T
}

Channel struct {
    bounds Bounds<float64>

    @go marshal
}
`),
		)

		It("Should pass a shipped type that only omitted fields reach", func() {
			write(root, v0Path, `
Status struct {
    message string
    variant string
}

Channel struct {
    key    uuid @key
    name   string = "a"
    status Status? {
        @go marshal omit
    }

    @go marshal
}
`)
			Expect(messages()).To(BeEmpty())
		})

		It("Should pass a member added to a stored enum", func() {
			const channel = `
import "schemas/x/versions/text/v0"

Channel struct {
    key   uuid @key
    level text.Level = h1

    @go marshal
}
`
			write(releaseRoot, v0Path, channel)
			write(root, v0Path, channel)
			write(releaseRoot, "schemas/x/versions/text/v0.oracle", `
Level enum {
    h1    = "h1"
    small = "small"
}
`)
			write(root, "schemas/x/versions/text/v0.oracle", `
Level enum {
    h1    = "h1"
    small = "small"
    tiny  = "tiny"
}
`)
			Expect(messages()).To(BeEmpty())
		})

		It(
			"Should pass a type that moves to another chain with the same shape",
			func() {
				const shippedOntology = `
ID struct {
    type string
    key  string

    @go marshal
}

Relationship struct {
    from ID
    to   ID

    @go hand
    @go marshal
}
`
				write(
					releaseRoot,
					"schemas/synnax/versions/ontology/v0.oracle",
					shippedOntology,
				)
				write(releaseRoot, v0Path, `
import "schemas/synnax/versions/ontology/v0"

Channel struct {
    key   uuid @key
    owner ontology.ID

    @go marshal
}
`)
				write(root, "schemas/synnax/versions/resource/v0.oracle", `
ID struct {
    type string
    key  string

    @go marshal
}
`)
				write(root, "schemas/synnax/versions/ontology/v0.oracle", `
import "schemas/synnax/versions/resource/v0"

Relationship struct {
    from resource.ID
    to   resource.ID

    @go hand
    @go marshal
}
`)
				write(root, v0Path, `
import "schemas/synnax/versions/resource/v0"

Channel struct {
    key   uuid @key
    owner resource.ID

    @go marshal
}
`)
				Expect(messages()).To(BeEmpty())
			},
		)

		It("Should pass a union replaced by another chain's union with more variants",
			func() {
				write(releaseRoot, v0Path, `
Scale union on type {
    linear {
        slope float64 = 1
    }
    none {}

    @go marshal
}

Channel struct {
    key   uuid @key
    scale Scale = none

    @go marshal
}
`)
				write(root, "schemas/synnax/versions/task/v0.oracle", `
Scale union on type {
    linear {
        slope float64 = 1
    }
    map {
        min float64 = 0 {
            @default group "range"
        }
        max float64 = 1 {
            @default group "range"
        }
    }
    none {}

    @go marshal
}
`)
				write(root, v0Path, `
import "schemas/synnax/versions/task/v0"

Channel struct {
    key   uuid @key
    scale task.Scale = none

    @go marshal
}
`)
				Expect(messages()).To(BeEmpty())
			})

		It("Should pass a shipped chain marked removed", func() {
			write(root, "schemas/synnax/versions/channel/removed", "")
			write(root, "schemas/synnax/versions/other/v0.oracle", shippedV0)
			Expect(messages()).To(BeEmpty())
		})
	})

	Describe("Rejected changes", func() {
		It("Should fail a changed default in a shipped version file", func() {
			write(releaseRoot, v0Path, `
RegisterValue struct {
    data_type string = "uint16"

    @go marshal
}

Channel union on type {
    holding_register extends RegisterValue {}

    @go marshal
}
`)
			write(root, v0Path, `
RegisterValue struct {
    data_type string = "uint8"

    @go marshal
}

Channel union on type {
    holding_register extends RegisterValue {}
    holding_register_bit {
        bit uint8 = 0
    }

    @go marshal
}
`)
			Expect(messages()).To(ConsistOf(
				shapeChange(
					"Channel.holding_register.data_type",
					"changes its default",
				),
				"schemas/synnax/channel v0 changes the stored shape of RegisterValue "+
					"(RegisterValue.data_type: changes its default), which "+
					"core/v0.60.1 shipped",
			))
		})

		It("Should fail a member removed from a stored enum", func() {
			write(releaseRoot, v0Path, `
Mode enum {
    read  = "read"
    write = "write"
}

Channel struct {
    key  uuid @key
    mode Mode = read

    @go marshal
}
`)
			write(root, v0Path, `
Mode enum {
    read = "read"
}

Channel struct {
    key  uuid @key
    mode Mode = read

    @go marshal
}
`)
			Expect(messages()).To(ConsistOf(
				shapeChange("Channel.mode", "member write removed"),
			))
		})

		It("Should fail a renumbered member of a stored enum", func() {
			write(releaseRoot, v0Path, `
Mode enum {
    read  = 1
    write = 2
}

Channel struct {
    key  uuid @key
    mode Mode

    @go marshal
}
`)
			write(root, v0Path, `
Mode enum {
    read  = 1
    write = 3
}

Channel struct {
    key  uuid @key
    mode Mode

    @go marshal
}
`)
			Expect(messages()).To(ConsistOf(
				shapeChange("Channel.mode", "member write changes its value"),
			))
		})

		It("Should fail a changed field type", func() {
			write(releaseRoot, v0Path, `
Channel struct {
    key  uuid @key
    rate float32

    @go marshal
}
`)
			write(root, v0Path, `
Channel struct {
    key  uuid @key
    rate float64

    @go marshal
}
`)
			Expect(messages()).To(ConsistOf(
				shapeChange("Channel.rate", "changes from float32 to float64"),
			))
		})

		It("Should fail a moved key", func() {
			write(root, v0Path, `
Status struct {
    message string
}

Channel struct {
    key    uuid
    name   string = "a" @key
    status Status? {
        @go marshal omit
    }

    @go marshal
}
`)
			Expect(messages()).To(ConsistOf(shapeChange("Channel.key", "changes @key")))
		})

		It("Should fail a changed optionality", func() {
			write(releaseRoot, v0Path, `
Channel struct {
    key  uuid @key
    unit string

    @go marshal
}
`)
			write(root, v0Path, `
Channel struct {
    key  uuid @key
    unit string?

    @go marshal
}
`)
			Expect(messages()).To(ConsistOf(
				shapeChange("Channel.unit", "changes optionality"),
			))
		})

		It("Should fail a union variant that is removed", func() {
			write(releaseRoot, v0Path, `
Channel union on type {
    analog {
        port string
    }
    digital {
        line uint8
    }

    @go marshal
}
`)
			write(root, v0Path, `
Channel union on type {
    analog {
        port string
    }

    @go marshal
}
`)
			Expect(messages()).To(ConsistOf(
				shapeChange("Channel.digital", "variant removed"),
			))
		})

		It("Should pass a stored base that moves to another chain", func() {
			write(releaseRoot, v0Path, `
Base struct {
    key uuid @key

    @go marshal
}

Channel struct extends Base {
    name string

    @go marshal
}
`)
			write(root, "schemas/synnax/versions/resource/v0.oracle", `
Base struct {
    key uuid @key

    @go marshal
}
`)
			write(root, v0Path, `
import "schemas/synnax/versions/resource/v0"

Channel struct extends resource.Base {
    name string

    @go marshal
}
`)
			Expect(messages()).To(BeEmpty())
		})

		It("Should pass a two-level stored base that moves to another chain", func() {
			write(releaseRoot, v0Path, `
Base0 struct {
    key uuid @key

    @go marshal
}

Base1 struct extends Base0 {
    name string

    @go marshal
}

Channel struct extends Base1 {
    rate float32

    @go marshal
}
`)
			write(root, "schemas/synnax/versions/resource/v0.oracle", `
Base0 struct {
    key uuid @key

    @go marshal
}

Base1 struct extends Base0 {
    name string

    @go marshal
}
`)
			write(root, v0Path, `
import "schemas/synnax/versions/resource/v0"

Channel struct extends resource.Base1 {
    rate float32

    @go marshal
}
`)
			Expect(messages()).To(BeEmpty())
		})

		It("Should fail a union variant that drops a base field", func() {
			write(releaseRoot, v0Path, `
Base struct {
    label  string
    hidden bool
}

Channel union on type extends Base {
    analog {
        port string
    }

    @go marshal
}
`)
			write(root, v0Path, `
Base struct {
    label  string
    hidden bool
}

Channel union on type extends Base {
    analog {
        -hidden
        port string
    }

    @go marshal
}
`)
			Expect(messages()).To(ConsistOf(shapeChange(
				"Channel.analog",
				"stored fields change from [label hidden port] to [label port]",
			)))
		})

		It("Should fail a type parameter with a new default", func() {
			write(releaseRoot, v0Path, `
Bounds struct<T extends numeric = float64> {
    lower T
}

Channel struct {
    bounds Bounds

    @go marshal
}
`)
			write(root, v0Path, `
Bounds struct<T extends numeric = float32> {
    lower T
}

Channel struct {
    bounds Bounds

    @go marshal
}
`)
			Expect(messages()).To(ConsistOf(
				shapeChange("Channel.bounds.lower", "changes from float64 to float32"),
			))
		})

		DescribeTable("Should fail a generic instance with a new filled-in shape",
			func(shipped, current, rel, reason string) {
				write(releaseRoot, v0Path, shipped)
				write(root, v0Path, current)
				Expect(messages()).To(ConsistOf(shapeChange(rel, reason)))
			},
			Entry("an alias whose type argument changes the shape", `
Inner struct {
    value uint8
}

Channel struct {
    payload Inner

    @go marshal
}
`, `
Real struct<T> {
    value T
}

Inner = Real<uint16>

Channel struct {
    payload Inner

    @go marshal
}
`, "Channel.payload.value", "changes from uint8 to uint16"),
			Entry("a nested generic instance with a new argument", `
Wrap struct<T> {
    item T
}

Box struct<T> {
    value T
}

Channel struct {
    payload Wrap<Box<uint8>>

    @go marshal
}
`, `
Wrap struct<T> {
    item T
}

Box struct<T> {
    value T
}

Channel struct {
    payload Wrap<Box<uint16>>

    @go marshal
}
`, "Channel.payload.item.value", "changes from uint8 to uint16"),
			Entry("an omitted type argument whose default differs", `
Bounds struct<T> {
    lower T
}

Channel struct {
    bounds Bounds<float64>

    @go marshal
}
`, `
Bounds struct<T = float32> {
    lower T
}

Channel struct {
    bounds Bounds

    @go marshal
}
`, "Channel.bounds.lower", "changes from float64 to float32"),
			Entry("a union base with a new type argument", `
Head struct<T> {
    unit T
}

Channel union on type extends Head<uint8> {
    none {}

    @go marshal
}
`, `
Head struct<T> {
    unit T
}

Channel union on type extends Head<uint16> {
    none {}

    @go marshal
}
`, "Channel.none.unit", "changes from uint8 to uint16"),
		)

		It("Should fail a removed stored type that only an omitted field reaches",
			func() {
				write(releaseRoot, v0Path, `
Status struct {
    message string

    @go marshal
}

Channel struct {
    key    uuid @key
    name   string = "a"
    status Status? {
        @go marshal omit
    }

    @go marshal
}
`)
				write(root, v0Path, `
Channel struct {
    key  uuid @key
    name string = "a"

    @go marshal
}
`)
				Expect(messages()).To(ConsistOf(
					"schemas/synnax/channel v0 removes Status, which core/v0.60.1 shipped",
				))
			})

		It("Should fail a stored field added to a shipped version file", func() {
			write(root, v0Path, `
Status struct {
    message string
}

Group struct {
    key uuid
}

Channel struct {
    key    uuid @key
    name   string = "a"
    group  Group
    status Status? {
        @go marshal omit
    }

    @go marshal
}
`)
			Expect(messages()).To(ConsistOf(shapeChange(
				"Channel", "stored fields change from [key name] to [key name group]",
			)))
		})

		It("Should fail a type that moves to another chain with a new shape", func() {
			write(releaseRoot, v0Path, `
ID struct {
    type string
    key  string

    @go marshal
}

Channel struct {
    key   uuid @key
    owner ID

    @go marshal
}
`)
			write(root, "schemas/synnax/versions/resource/v0.oracle", `
ID struct {
    key string

    @go marshal
}
`)
			write(root, v0Path, `
import "schemas/synnax/versions/resource/v0"

Channel struct {
    key   uuid @key
    owner resource.ID

    @go marshal
}
`)
			Expect(messages()).To(ConsistOf(shapeChange(
				"Channel.owner", "stored fields change from [type key] to [key]",
			)))
		})

		It("Should fail a stored type that nothing else reaches when it is removed",
			func() {
				write(releaseRoot, v0Path, shippedV0+`
Group struct {
    key uuid @key

    @go marshal
}
`)
				write(root, v0Path, shippedV0)
				Expect(messages()).To(ConsistOf(
					"schemas/synnax/channel v0 removes Group, which core/v0.60.1 shipped",
				))
			})

		It("Should fail a shipped alias line that now resolves to a new shape", func() {
			write(releaseRoot, v1Path, "Channel = v0.Channel\n")
			write(root, v0Path, shippedV0)
			write(root, v1Path, `
Channel struct {
    key  uuid @key
    name string = "c"

    @go marshal
}
`)
			Expect(messages()).To(ContainElement(
				"schemas/synnax/channel v1 changes the stored shape of Channel " +
					"(Channel.name: changes its default), which core/v0.60.1 shipped",
			))
		})

		It("Should fail every shipped version file when no chain remains", func() {
			write(root, "schemas/synnax/channel.oracle", shippedV0)
			Expect(messages()).To(ConsistOf(
				"schemas/synnax/channel v0 is gone, which core/v0.60.1 shipped",
			))
		})

		It("Should fail a shipped version file that is gone", func() {
			write(root, "schemas/synnax/versions/other/v0.oracle", shippedV0)
			Expect(messages()).To(ContainElement(
				"schemas/synnax/channel v0 is gone, which core/v0.60.1 shipped",
			))
		})

		It("Should fail a removed chain that a remaining stored type imports", func() {
			const holder = `
import "schemas/synnax/versions/channel/v0"

Holder struct {
    key     uuid @key
    channel channel.Channel

    @go marshal
}
`
			write(root, "schemas/synnax/versions/channel/removed", "")
			write(releaseRoot, "schemas/synnax/versions/holder/v0.oracle", holder)
			write(root, "schemas/synnax/versions/holder/v0.oracle", holder)
			Expect(messages()).To(ContainElement(
				"no version chain for schemas/synnax/channel",
			))
		})

		DescribeTable("Should fail a changed stored shape",
			func(shipped, current, reason string) {
				write(releaseRoot, v0Path, shipped)
				write(root, v0Path, current)
				Expect(messages()).To(ContainElement(ContainSubstring(reason)))
			},
			Entry("a type that changes its form", `
Mode uint8

Channel struct {
    mode Mode

    @go marshal
}
`, `
Mode enum {
    a = 1
}

Channel struct {
    mode Mode

    @go marshal
}
`, "Channel.mode: changes from uint8 to an enum"),
			Entry("a distinct type with a new base", `
Mode uint8

Channel struct {
    mode Mode

    @go marshal
}
`, `
Mode uint16

Channel struct {
    mode Mode

    @go marshal
}
`, "Channel.mode: changes from uint8 to uint16"),
			Entry("an alias with a new target", `
Inner struct {
    a string
}

Payload = Inner

Channel struct {
    payload Payload

    @go marshal
}
`, `
Inner struct {
    a string
}

Other struct {
    a uint8
}

Payload = Other

Channel struct {
    payload Payload

    @go marshal
}
`, "Channel.payload.a: changes from string to uint8"),
			Entry("a map with a new value type", `
Channel struct {
    labels map<string, string>

    @go marshal
}
`, `
Channel struct {
    labels map<string, uint8>

    @go marshal
}
`, "Channel.labels: changes from string to uint8"),
			Entry("a map that becomes an array", `
Channel struct {
    labels map<string, string>

    @go marshal
}
`, `
Channel struct {
    labels string[]

    @go marshal
}
`, "Channel.labels: changes from Map to Array"),
			Entry("a fixed array with a new size", `
Channel struct {
    bytes uint8[4]

    @go marshal
}
`, `
Channel struct {
    bytes uint8[8]

    @go marshal
}
`, "Channel.bytes: changes its array size"),
			Entry("a generic type with new type parameters", `
Box struct<T> {
    value T

    @go marshal
}
`, `
Box struct<T, U> {
    value T

    @go marshal
}
`, "Box: changes its type parameters"),
			Entry("a field that drops its type parameter", `
Box struct<T> {
    value T

    @go marshal
}
`, `
Box struct<T> {
    value string

    @go marshal
}
`, "Box.value: changes from type parameter 1 to string"),
			Entry("a generic type that swaps its type parameters", `
Pair struct<A, B> {
    a A
    b B

    @go marshal
}
`, `
Pair struct<A, B> {
    a B
    b A

    @go marshal
}
`, "Pair.a: changes from type parameter 1 to type parameter 2"),
			Entry("a generic alias that binds a concrete type", `
Inner struct<T> {
    value T
}

Payload<T> = Inner<T>

Channel struct {
    payload Payload<uint8>

    @go marshal
}
`, `
Inner struct<T> {
    value T
}

Payload<T> = Inner<string>

Channel struct {
    payload Payload<uint8>

    @go marshal
}
`, "Channel.payload.value: changes from uint8 to string"),
			Entry("a generic alias that swaps its type parameters", `
Pair struct<A, B> {
    a A
    b B
}

Swapped<A, B> = Pair<A, B>

Channel struct {
    pair Swapped<string, uint8>

    @go marshal
}
`, `
Pair struct<A, B> {
    a A
    b B
}

Swapped<A, B> = Pair<B, A>

Channel struct {
    pair Swapped<string, uint8>

    @go marshal
}
`, "Channel.pair.a: changes from string to uint8"),
			Entry("a generic alias on one side that binds a concrete type", `
Inner struct<T> {
    value T
}

Channel struct {
    payload Inner<uint8>

    @go marshal
}
`, `
Real struct<T> {
    value T
}

Inner<T> = Real<string>

Channel struct {
    payload Inner<uint8>

    @go marshal
}
`, "Channel.payload.value: changes from uint8 to string"),
			Entry("a generic alias on one side that swaps its type parameters", `
Inner struct<A, B> {
    a A
    b B
}

Channel struct {
    payload Inner<uint8, string>

    @go marshal
}
`, `
Real struct<A, B> {
    a A
    b B
}

Inner<A, B> = Real<B, A>

Channel struct {
    payload Inner<uint8, string>

    @go marshal
}
`, "Channel.payload.a: changes from uint8 to string"),
			Entry("a stored root replaced by an alias with a new shape", `
Channel struct {
    value uint8

    @go marshal
}
`, `
Real struct<T> {
    value T
}

Channel = Real<string>
`, "Channel.value: changes from uint8 to string"),
			Entry("a field that drops its nil type argument", `
Status struct<D> {
    details D?
}

Channel struct {
    status Status<nil>

    @go marshal
}
`, `
Status struct<D> {
    details D?
}

Channel struct {
    status Status<string>

    @go marshal
}
`, "Channel.status.details: changes from nil to string"),
			Entry("an enum that changes between int and string members", `
Level enum {
    a = 1
}

Channel struct {
    level Level

    @go marshal
}
`, `
Level enum {
    a = "a"
}

Channel struct {
    level Level

    @go marshal
}
`, "Channel.level: changes between int and string members"),
			Entry("a union with a new discriminator", `
Scale union on type {
    none {}

    @go marshal
}
`, `
Scale union on kind {
    none {}

    @go marshal
}
`, "Scale: changes its discriminator"),
			Entry("a union that starts to borrow an enum", `
Scale union on type {
    none {}

    @go marshal
}
`, `
Type enum {
    none = "none"
}

Scale union on type Type {
    none {}

    @go marshal
}
`, "Scale: changes its discriminator type"),
			Entry("a union whose borrowed enum renames a member", `
Type enum {
    none = "none"
}

Scale union on type Type {
    none {}

    @go marshal
}
`, `
Type enum {
    off = "none"
}

Scale union on type Type {
    none {}

    @go marshal
}
`, "Scale: member none removed"),
			Entry("a union that gains a base", `
Base struct {
    unit string
}

Scale union on type {
    none {}

    @go marshal
}
`, `
Base struct {
    unit string
}

Scale union on type extends Base {
    none {}

    @go marshal
}
`, "Scale.none: stored fields change from [] to [unit]"),
			Entry("a union whose base gains a field", `
Base struct {
    unit string
}

Scale union on type extends Base {
    none {}

    @go marshal
}
`, `
Base struct {
    unit   string
    offset float64
}

Scale union on type extends Base {
    none {}

    @go marshal
}
`, "Scale.none: stored fields change from [unit] to [unit offset]"),
			Entry("a union variant that becomes named", `
Scale union on type {
    linear {
        slope float64
    }

    @go marshal
}
`, `
Linear struct {
    slope float64
}

Scale union on type {
    linear Linear

    @go marshal
}
`, "Scale.linear: changes between inline and named"),
		)

		It("Should fail a release whose schemas directory is malformed", func() {
			write(root, v0Path, shippedV0)
			write(releaseRoot, "schemas/synnax/versions/README.md", "")
			Expect(messages()).To(ConsistOf(
				ContainSubstring("must contain only per-resource chain directories"),
			))
		})

		It("Should fail a shipped version file that no longer resolves", func() {
			write(releaseRoot, v0Path, `
import "schemas/x/versions/text/v0"

Channel struct {
    level text.Level

    @go marshal
}
`)
			write(root, v0Path, shippedV0)
			Expect(messages()).To(ContainElement(
				ContainSubstring(v0Path + " at core/v0.60.1"),
			))
		})

		It("Should fail a current version file that does not resolve", func() {
			write(root, v0Path, `
import "schemas/x/versions/text/v0"

Channel struct {
    level text.Level

    @go marshal
}
`)
			Expect(messages()).To(ContainElement(ContainSubstring("text")))
		})
	})

	It("Should pass a generic stored type that renames its type parameter", func() {
		write(releaseRoot, v0Path, `
Box struct<T> {
    value T

    @go marshal
}
`)
		write(root, v0Path, `
Box struct<U> {
    value U

    @go marshal
}
`)
		Expect(messages()).To(BeEmpty())
	})

	It("Should pass a stored field with a nil type argument", func() {
		const channel = `
Status struct<D> {
    details D?
}

Channel struct {
    status Status<nil>

    @go marshal
}
`
		write(releaseRoot, v0Path, channel)
		write(root, v0Path, channel)
		Expect(messages()).To(BeEmpty())
	})

	It("Should pass a base whose omitted type argument takes its default", func() {
		const channel = `
Bounds struct<T extends numeric = float64> {
    lower T
}

Channel struct extends Bounds {
    name string

    @go marshal
}
`
		write(releaseRoot, v0Path, channel)
		write(root, v0Path, channel)
		Expect(messages()).To(BeEmpty())
	})

	It("Should fail a base whose omitted type argument takes a new default", func() {
		write(releaseRoot, v0Path, `
Bounds struct<T extends numeric = float64> {
    lower T
}

Channel struct extends Bounds {
    @go marshal
}
`)
		write(root, v0Path, `
Bounds struct<T extends numeric = int32> {
    lower T
}

Channel struct extends Bounds {
    @go marshal
}
`)
		Expect(messages()).To(ContainElement(ContainSubstring("Channel.lower")))
	})

	It("Should fail a changed field in a type that only migrates", func() {
		write(releaseRoot, v0Path, `
Channel struct {
    name string

    @go migrate
}
`)
		write(root, v0Path, `
Channel struct {
    name int32

    @go migrate
}
`)
		Expect(messages()).To(ContainElement(ContainSubstring("Channel.name")))
	})

	It("Should fail a new base default that a generic struct does not shadow", func() {
		write(releaseRoot, v0Path, `
Bounds struct<T extends numeric = float64> {
    lower T
}

Channel struct<T> extends Bounds {
    value T

    @go marshal
}
`)
		write(root, v0Path, `
Bounds struct<T extends numeric = int32> {
    lower T
}

Channel struct<T> extends Bounds {
    value T

    @go marshal
}
`)
		Expect(messages()).To(ContainElement(ContainSubstring("Channel.lower")))
	})

	It("Should pass a recursive stored type that keeps its shape", func() {
		const node = `
Node struct {
    name string
    next Node?

    @go marshal
}
`
		write(releaseRoot, v0Path, node)
		write(root, v0Path, node)
		Expect(messages()).To(BeEmpty())
	})
})
