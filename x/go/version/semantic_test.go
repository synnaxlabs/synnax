package version_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/version"
)

var _ = Describe("Semantic", func() {
	Describe("ParseSemVer", func() {
		It("should parse valid semantic versions", func() {
			major, minor, patch, err := version.ParseSemVer("1.2.3")
			Expect(err).NotTo(HaveOccurred())
			Expect(major).To(Equal(1))
			Expect(minor).To(Equal(2))
			Expect(patch).To(Equal(3))
		})

		It("should return error for invalid format", func() {
			_, _, _, err := version.ParseSemVer("1.2")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid semver format"))
		})

		It("should return error for non-numeric major version", func() {
			_, _, _, err := version.ParseSemVer("a.2.3")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid major version"))
		})

		It("should return error for non-numeric minor version", func() {
			_, _, _, err := version.ParseSemVer("1.b.3")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid minor version"))
		})

		It("should return error for non-numeric patch version", func() {
			_, _, _, err := version.ParseSemVer("1.2.c")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid patch version"))
		})
	})

	Describe("CompareSemantic", func() {
		It("should compare versions correctly", func() {
			tests := []struct {
				a, b     version.Semantic
				expected int
			}{
				{"1.2.3", "1.2.3", 0},
				{"1.2.3", "1.2.4", -1},
				{"1.2.4", "1.2.3", 1},
				{"1.3.0", "1.2.9", 1},
				{"2.0.0", "1.9.9", 1},
				{"1.9.9", "2.0.0", -1},
				{"1.0.0", "2.0.0", -1},
				{"1.0.0", "1.1.0", -1},
				{"1.0.0", "1.0.1", -1},
			}

			for _, test := range tests {
				result, err := version.CompareSemantic(test.a, test.b)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(test.expected),
					"comparing %s to %s", test.a, test.b)
			}
		})

		It("should handle comparison options", func() {
			tests := []struct {
				a, b     version.Semantic
				opts     []version.CompareSemanticOption
				expected int
			}{
				{"1.2.3", "2.2.3", []version.CompareSemanticOption{version.WithSkipMajor()}, 0},
				{"1.2.3", "1.3.3", []version.CompareSemanticOption{version.WithSkipMinor()}, 0},
				{"1.2.3", "1.2.4", []version.CompareSemanticOption{version.WithSkipPatch()}, 0},
			}

			for _, test := range tests {
				result, err := version.CompareSemantic(test.a, test.b, test.opts...)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(test.expected))
			}
		})

		It("should return error for invalid versions", func() {
			_, err := version.CompareSemantic("1.2", "1.2.3")
			Expect(err).To(HaveOccurred())

			_, err = version.CompareSemantic("1.2.3", "1.2")
			Expect(err).To(HaveOccurred())
		})
	})
})
