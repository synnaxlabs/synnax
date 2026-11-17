// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("StandardFileLoader", func() {
	var (
		repoRoot string
		loader   analyzer.FileLoader
	)
	BeforeEach(func() {
		repoRoot = GinkgoT().TempDir()
		loader = analyzer.NewStandardFileLoader(repoRoot)
	})

	It("Should load a schema by its repo-relative import path", func() {
		Expect(os.WriteFile(filepath.Join(repoRoot, "a.oracle"), []byte("A"), 0o644)).
			To(Succeed())
		content, path := MustSucceed2(loader.Load("a"))
		Expect(content).To(Equal("A"))
		Expect(path).To(Equal("a.oracle"))
	})

	It("Should fail for an import path that leads out of the repository", func() {
		Expect(loader.Load("../a")).Error().To(MatchError(ContainSubstring("escapes")))
	})
})
