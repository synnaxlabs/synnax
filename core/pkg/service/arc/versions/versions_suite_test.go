// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	graph "github.com/synnaxlabs/arc/graph/versions/v1"
	text "github.com/synnaxlabs/arc/text/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	. "github.com/synnaxlabs/x/testutil"
)

func TestArcVersions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Arc Versions Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// textOf returns the text of the text-mode Arc b, failing the spec otherwise.
func textOf(b versions.Base) text.Text {
	GinkgoHelper()
	src, ok := b.Source.Variant.(versions.TextSource)
	Expect(ok).To(BeTrue(), "expected a text source, got %T", b.Source.Variant)
	return src.Text
}

// graphOf returns the graph of the graph-mode Arc b, failing the spec otherwise.
func graphOf(b versions.Base) graph.Graph {
	GinkgoHelper()
	src, ok := b.Source.Variant.(versions.GraphSource)
	Expect(ok).To(BeTrue(), "expected a graph source, got %T", b.Source.Variant)
	return src.Graph
}
