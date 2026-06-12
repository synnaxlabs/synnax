// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"
	"go/parser"
	"go/token"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
	xtestutil "github.com/synnaxlabs/x/testutil"
)

// generateRequest creates a plugin request from a source string by analyzing the source
// and returning a request with the resolution table.
func generateRequest(
	ctx context.Context,
	source string,
	namespace string,
	loader *MockFileLoader,
) (*plugin.Request, error) {
	table, diag := analyzer.AnalyzeSource(ctx, source, namespace, loader)
	if diag != nil && !diag.Ok() {
		errors := diag.Errors()
		if len(errors) > 0 {
			return nil, diag
		}
	}
	return &plugin.Request{
		Resolutions: table,
		RepoRoot:    loader.RepoRoot(),
	}, nil
}

// MustGenerateRequest is like GenerateRequest but fails the test if analysis fails.
func MustGenerateRequest(
	ctx context.Context,
	source string,
	namespace string,
	loader *MockFileLoader,
) *plugin.Request {
	ginkgo.GinkgoHelper()
	req, err := generateRequest(ctx, source, namespace, loader)
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "failed to analyze source")
	return req
}

// MustGenerateMulti analyzes every file registered on the loader as one
// resolution table and runs the provided plugin against it. Use it for tests
// that need to exercise cross-schema behavior (e.g., two schemas that derive
// the same namespace from different files).
func MustGenerateMulti(
	ctx context.Context,
	loader *MockFileLoader,
	p plugin.Plugin,
) *plugin.Response {
	ginkgo.GinkgoHelper()
	files := make([]string, 0, len(loader.Files))
	for path := range loader.Files {
		files = append(files, path)
	}
	table, diag := analyzer.Analyze(ctx, files, loader)
	if diag != nil && !diag.Ok() {
		gomega.Expect(diag.Errors()).To(gomega.BeEmpty(), "failed to analyze sources")
	}
	req := &plugin.Request{Resolutions: table, RepoRoot: loader.RepoRoot()}
	resp, err := p.Generate(req)
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "failed to generate")
	return resp
}

// MustGenerate analyzes the source and generates output using the provided plugin.
// Fails the test if analysis or generation fails.
func MustGenerate(
	ctx context.Context,
	source string,
	namespace string,
	loader *MockFileLoader,
	p plugin.Plugin,
) *plugin.Response {
	ginkgo.GinkgoHelper()
	req := MustGenerateRequest(ctx, source, namespace, loader)
	resp, err := p.Generate(req)
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "failed to generate")
	return resp
}

// contentOf finds a file in the response by path suffix and returns its content.
// Returns empty string if no matching file is found.
func contentOf(resp *plugin.Response, pathSuffix string) string {
	for _, f := range resp.Files {
		if strings.HasSuffix(f.Path, pathSuffix) {
			return string(f.Content)
		}
	}
	return ""
}

// MustContentOf is like ContentOf but fails the test if no matching file is found.
func MustContentOf(resp *plugin.Response, pathSuffix string) string {
	ginkgo.GinkgoHelper()
	content := contentOf(resp, pathSuffix)
	gomega.Expect(content).NotTo(gomega.BeEmpty(), "no file found with suffix: %s", pathSuffix)
	return content
}

// ContentExpectation provides fluent assertions for generated content.
type ContentExpectation struct {
	content string
}

// ExpectContent creates a ContentExpectation for fluent assertions.
// Fails if no file with the given suffix is found.
func ExpectContent(resp *plugin.Response, pathSuffix string) *ContentExpectation {
	ginkgo.GinkgoHelper()
	content := MustContentOf(resp, pathSuffix)
	return &ContentExpectation{content: content}
}

// ToContain asserts that the content contains all the given substrings.
func (c *ContentExpectation) ToContain(substrings ...string) *ContentExpectation {
	ginkgo.GinkgoHelper()
	for _, s := range substrings {
		gomega.Expect(c.content).To(gomega.ContainSubstring(s), "expected content to contain: %q", s)
	}
	return c
}

// ToBeValidGoSource asserts that the content parses as a syntactically valid
// Go source file. Substring assertions cannot catch malformed emission around
// the asserted fragments; this can.
func (c *ContentExpectation) ToBeValidGoSource() *ContentExpectation {
	ginkgo.GinkgoHelper()
	xtestutil.MustSucceed(parser.ParseFile(
		token.NewFileSet(), "", c.content, parser.SkipObjectResolution))
	return c
}

// ToNotContain asserts that the content does not contain any of the given substrings.
func (c *ContentExpectation) ToNotContain(substrings ...string) *ContentExpectation {
	ginkgo.GinkgoHelper()
	for _, s := range substrings {
		gomega.Expect(c.content).NotTo(gomega.ContainSubstring(s), "expected content to NOT contain: %q", s)
	}
	return c
}

// ToPreserveOrder asserts that the given substrings appear in order in the content.
func (c *ContentExpectation) ToPreserveOrder(orderedSubstrings ...string) *ContentExpectation {
	ginkgo.GinkgoHelper()
	lastIdx := -1
	for _, s := range orderedSubstrings {
		idx := strings.Index(c.content, s)
		gomega.Expect(idx).To(gomega.BeNumerically(">=", 0), "expected content to contain: %q", s)
		gomega.Expect(idx).To(gomega.BeNumerically(">", lastIdx),
			"expected %q to appear after previous substring", s)
		lastIdx = idx
	}
	return c
}
