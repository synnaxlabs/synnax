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

	BeforeEach(func() {
		root, releaseRoot = GinkgoT().TempDir(), GinkgoT().TempDir()
		write(releaseRoot, v0Path, shippedV0)
	})

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

	It("Should fail a changed default in a shipped version file", func() {
		write(root, v0Path, `
Status struct {
    message string
}

Channel struct {
    key    uuid @key
    name   string = "b"
    status Status? {
        @go marshal omit
    }

    @go marshal
}
`)
		Expect(messages()).To(ConsistOf(
			"schemas/synnax/channel v0 changes the shape of Channel, which " +
				"core/v0.60.1 shipped",
		))
	})

	It("Should fail a stored declaration added to a shipped version file", func() {
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
		Expect(messages()).To(ConsistOf(
			"schemas/synnax/channel v0 adds Group, which core/v0.60.1 shipped",
			"schemas/synnax/channel v0 changes the shape of Channel, which "+
				"core/v0.60.1 shipped",
		))
	})

	It("Should fail a shipped alias line that changes its target", func() {
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
			"schemas/synnax/channel v1 changes the alias line of Channel, which " +
				"core/v0.60.1 shipped",
		))
	})

	It("Should fail a shipped version file that is gone", func() {
		write(root, "schemas/synnax/versions/other/v0.oracle", shippedV0)
		Expect(messages()).To(ContainElement(
			"schemas/synnax/channel v0 is gone, which core/v0.60.1 shipped",
		))
	})
})
