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

		It("Should fail a shipped version file that is gone", func() {
			write(root, "schemas/synnax/versions/other/v0.oracle", shippedV0)
			Expect(messages()).To(ContainElement(
				"schemas/synnax/channel v0 is gone, which core/v0.60.1 shipped",
			))
		})
	})
})
