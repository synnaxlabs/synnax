// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package copyright_test

import (
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/copyright"
)

func TestCopyright(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Copyright Suite")
}

var _ = Describe("Copyright", func() {
	currentYear := time.Now().Year()

	Describe("Ensure", func() {
		It("should add copyright header to content without one", func() {
			content := `import "schemas/label"

Status struct {
    key uuid
}
`
			result, err := copyright.Ensure(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HavePrefix("// Copyright"))
			Expect(result).To(ContainSubstring("Synnax Labs"))
			Expect(result).To(ContainSubstring(`import "schemas/label"`))
		})

		It("should preserve content that already has current year copyright", func() {
			content := `// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "schemas/label"

Status struct {
    key uuid
}
`
			result, err := copyright.Ensure(content)
			Expect(err).ToNot(HaveOccurred())
			// Should be unchanged if year matches
			if currentYear == 2025 {
				Expect(result).To(Equal(content))
			}
		})

		It("should update outdated year in existing copyright", func() {
			content := `// Copyright 2020 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "schemas/label"
`
			result, err := copyright.Ensure(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring("Copyright 2025"))
			Expect(result).ToNot(ContainSubstring("Copyright 2020"))
			Expect(result).To(ContainSubstring(`import "schemas/label"`))
		})

		It("should handle content with only struct definition", func() {
			content := `Status struct {
    key uuid
}
`
			result, err := copyright.Ensure(content)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HavePrefix("// Copyright"))
			Expect(strings.Count(result, "Status struct")).To(Equal(1))
		})
	})
})
