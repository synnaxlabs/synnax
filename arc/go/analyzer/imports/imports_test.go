// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imports_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/text"
)

func TestImports(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Imports Suite")
}

func hasErrors(diag *diagnostics.Diagnostics) bool {
	if diag == nil {
		return false
	}
	for _, d := range *diag {
		if d.Severity == diagnostics.Error {
			return true
		}
	}
	return false
}

func hasWarnings(diag *diagnostics.Diagnostics) bool {
	if diag == nil {
		return false
	}
	for _, d := range *diag {
		if d.Severity == diagnostics.Warning {
			return true
		}
	}
	return false
}

var _ = Describe("Imports", func() {
	Describe("Basic Import Parsing", func() {
		It("Should parse a valid import block", func() {
			src := `
import (
	math
)

func main() f64 {
	return 1.0
}
`
			t, diag := text.Parse(text.Text{Raw: src})
			Expect(diag).To(BeNil())
			_, diag = text.Analyze(context.Background(), t, nil)
			Expect(hasErrors(diag)).To(BeFalse())
			// math is imported but not used - should have warning
			Expect(hasWarnings(diag)).To(BeTrue())
		})

		It("Should reject unknown modules", func() {
			src := `
import (
	unknown_module
)

func main() f64 {
	return 1.0
}
`
			t, diag := text.Parse(text.Text{Raw: src})
			Expect(diag).To(BeNil())
			_, diag = text.Analyze(context.Background(), t, nil)
			Expect(hasErrors(diag)).To(BeTrue())
		})

		It("Should reject duplicate imports", func() {
			src := `
import (
	math
	math
)

func main() f64 {
	return 1.0
}
`
			t, diag := text.Parse(text.Text{Raw: src})
			Expect(diag).To(BeNil())
			_, diag = text.Analyze(context.Background(), t, nil)
			Expect(hasErrors(diag)).To(BeTrue())
		})
	})

	Describe("Module Member Access", func() {
		It("Should resolve math.sqrt", func() {
			src := `
import (
	math
)

func main() f64 {
	x := math.sqrt(16.0)
	return x
}
`
			t, diag := text.Parse(text.Text{Raw: src})
			Expect(diag).To(BeNil())
			_, diag = text.Analyze(context.Background(), t, nil)
			Expect(hasErrors(diag)).To(BeFalse())
			Expect(hasWarnings(diag)).To(BeFalse()) // math is used
		})

		It("Should reject unknown module members", func() {
			src := `
import (
	math
)

func main() f64 {
	x := math.unknown_function(16.0)
	return x
}
`
			t, diag := text.Parse(text.Text{Raw: src})
			Expect(diag).To(BeNil())
			_, diag = text.Analyze(context.Background(), t, nil)
			Expect(hasErrors(diag)).To(BeTrue())
		})
	})

	Describe("Multiple Imports", func() {
		It("Should handle multiple imports", func() {
			src := `
import (
	math
	time
)

func main() f64 {
	x := math.sqrt(16.0)
	return x
}
`
			t, diag := text.Parse(text.Text{Raw: src})
			Expect(diag).To(BeNil())
			_, diag = text.Analyze(context.Background(), t, nil)
			Expect(hasErrors(diag)).To(BeFalse())
			// time is imported but not used
			Expect(hasWarnings(diag)).To(BeTrue())
		})
	})
})
