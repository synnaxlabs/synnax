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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/plugin"
)

// GenerateRequest creates a plugin request from a source string by analyzing the source
// and returning a request with the resolution table.
func GenerateRequest(
	ctx context.Context,
	source string,
	namespace string,
	loader *MockFileLoader,
) (*plugin.Request, error) {
	table, diag := analyzer.AnalyzeSource(ctx, source, namespace, loader)
	if diag.HasErrors() {
		errors := diag.Errors()
		if len(errors) > 0 {
			return nil, errors
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
	GinkgoHelper()
	req, err := GenerateRequest(ctx, source, namespace, loader)
	Expect(err).To(BeNil(), "failed to analyze source")
	return req
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
	GinkgoHelper()
	req := MustGenerateRequest(ctx, source, namespace, loader)
	resp, err := p.Generate(req)
	Expect(err).To(BeNil(), "failed to generate")
	return resp
}

// ContentOf finds a file in the response by path suffix and returns its content.
// Returns empty string if no matching file is found.
func ContentOf(resp *plugin.Response, pathSuffix string) string {
	for _, f := range resp.Files {
		if strings.HasSuffix(f.Path, pathSuffix) {
			return string(f.Content)
		}
	}
	return ""
}

// MustContentOf is like ContentOf but fails the test if no matching file is found.
func MustContentOf(resp *plugin.Response, pathSuffix string) string {
	GinkgoHelper()
	content := ContentOf(resp, pathSuffix)
	Expect(content).NotTo(BeEmpty(), "no file found with suffix: %s", pathSuffix)
	return content
}

// ContentExpectation provides fluent assertions for generated content.
type ContentExpectation struct {
	content string
}

// ExpectContent creates a ContentExpectation for fluent assertions.
// Fails if no file with the given suffix is found.
func ExpectContent(resp *plugin.Response, pathSuffix string) *ContentExpectation {
	GinkgoHelper()
	content := MustContentOf(resp, pathSuffix)
	return &ContentExpectation{content: content}
}

// ToContain asserts that the content contains all the given substrings.
func (c *ContentExpectation) ToContain(substrings ...string) *ContentExpectation {
	GinkgoHelper()
	for _, s := range substrings {
		Expect(c.content).To(ContainSubstring(s), "expected content to contain: %q", s)
	}
	return c
}

// ToNotContain asserts that the content does not contain any of the given substrings.
func (c *ContentExpectation) ToNotContain(substrings ...string) *ContentExpectation {
	GinkgoHelper()
	for _, s := range substrings {
		Expect(c.content).NotTo(ContainSubstring(s), "expected content to NOT contain: %q", s)
	}
	return c
}

// ToPreserveOrder asserts that the given substrings appear in order in the content.
func (c *ContentExpectation) ToPreserveOrder(orderedSubstrings ...string) *ContentExpectation {
	GinkgoHelper()
	lastIdx := -1
	for _, s := range orderedSubstrings {
		idx := strings.Index(c.content, s)
		Expect(idx).To(BeNumerically(">=", 0), "expected content to contain: %q", s)
		Expect(idx).To(BeNumerically(">", lastIdx),
			"expected %q to appear after previous substring", s)
		lastIdx = idx
	}
	return c
}

// ToMatchRegexp asserts that the content matches the given regular expression.
func (c *ContentExpectation) ToMatchRegexp(pattern string) *ContentExpectation {
	GinkgoHelper()
	Expect(c.content).To(MatchRegexp(pattern))
	return c
}

// ToNotMatchRegexp asserts that the content does not match the given regular expression.
func (c *ContentExpectation) ToNotMatchRegexp(pattern string) *ContentExpectation {
	GinkgoHelper()
	Expect(c.content).NotTo(MatchRegexp(pattern))
	return c
}

// Content returns the raw content for custom assertions.
func (c *ContentExpectation) Content() string {
	return c.content
}

// PluginTestSetup provides common setup for plugin tests.
type PluginTestSetup struct {
	Ctx    context.Context
	Loader *MockFileLoader
}

// NewPluginTestSetup creates a new test setup with default values.
func NewPluginTestSetup() *PluginTestSetup {
	return &PluginTestSetup{
		Ctx:    context.Background(),
		Loader: NewMockFileLoader(),
	}
}

// Reset resets the test setup for a new test.
func (s *PluginTestSetup) Reset() {
	s.Ctx = context.Background()
	s.Loader = NewMockFileLoader()
}
