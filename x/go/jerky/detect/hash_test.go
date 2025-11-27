// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package detect_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/jerky/detect"
	"github.com/synnaxlabs/x/jerky/parse"
	"github.com/synnaxlabs/x/jerky/state"
)

var _ = Describe("Hash", func() {
	Describe("ComputeStructHash", func() {
		It("should produce consistent hash for same struct", func() {
			parsed := parse.ParsedStruct{
				Name: "User",
				Fields: []parse.ParsedField{
					{Name: "Name", GoType: parse.GoType{Name: "string"}},
					{Name: "Age", GoType: parse.GoType{Name: "int"}},
				},
			}

			hash1 := detect.ComputeStructHash(parsed)
			hash2 := detect.ComputeStructHash(parsed)
			Expect(hash1).To(Equal(hash2))
		})

		It("should produce different hash when field added", func() {
			original := parse.ParsedStruct{
				Name: "User",
				Fields: []parse.ParsedField{
					{Name: "Name", GoType: parse.GoType{Name: "string"}},
				},
			}

			modified := parse.ParsedStruct{
				Name: "User",
				Fields: []parse.ParsedField{
					{Name: "Name", GoType: parse.GoType{Name: "string"}},
					{Name: "Age", GoType: parse.GoType{Name: "int"}},
				},
			}

			hash1 := detect.ComputeStructHash(original)
			hash2 := detect.ComputeStructHash(modified)
			Expect(hash1).ToNot(Equal(hash2))
		})

		It("should produce different hash when field type changes", func() {
			original := parse.ParsedStruct{
				Name: "User",
				Fields: []parse.ParsedField{
					{Name: "Age", GoType: parse.GoType{Name: "int"}},
				},
			}

			modified := parse.ParsedStruct{
				Name: "User",
				Fields: []parse.ParsedField{
					{Name: "Age", GoType: parse.GoType{Name: "int64"}},
				},
			}

			hash1 := detect.ComputeStructHash(original)
			hash2 := detect.ComputeStructHash(modified)
			Expect(hash1).ToNot(Equal(hash2))
		})

		It("should produce different hash when field order changes", func() {
			original := parse.ParsedStruct{
				Name: "User",
				Fields: []parse.ParsedField{
					{Name: "Name", GoType: parse.GoType{Name: "string"}},
					{Name: "Age", GoType: parse.GoType{Name: "int"}},
				},
			}

			reordered := parse.ParsedStruct{
				Name: "User",
				Fields: []parse.ParsedField{
					{Name: "Age", GoType: parse.GoType{Name: "int"}},
					{Name: "Name", GoType: parse.GoType{Name: "string"}},
				},
			}

			hash1 := detect.ComputeStructHash(original)
			hash2 := detect.ComputeStructHash(reordered)
			Expect(hash1).ToNot(Equal(hash2))
		})

		It("should include struct name in hash", func() {
			struct1 := parse.ParsedStruct{
				Name: "User",
				Fields: []parse.ParsedField{
					{Name: "Name", GoType: parse.GoType{Name: "string"}},
				},
			}

			struct2 := parse.ParsedStruct{
				Name: "Admin",
				Fields: []parse.ParsedField{
					{Name: "Name", GoType: parse.GoType{Name: "string"}},
				},
			}

			hash1 := detect.ComputeStructHash(struct1)
			hash2 := detect.ComputeStructHash(struct2)
			Expect(hash1).ToNot(Equal(hash2))
		})
	})

	Describe("ComputeStructHashFromFields", func() {
		It("should match ParsedStruct hash when field order provided", func() {
			parsed := parse.ParsedStruct{
				Name: "User",
				Fields: []parse.ParsedField{
					{Name: "Name", GoType: parse.GoType{Name: "string"}, Tags: parse.StructTags{JSON: "name"}},
					{Name: "Age", GoType: parse.GoType{Name: "int"}, Tags: parse.StructTags{JSON: "age"}},
				},
			}

			fields := map[string]state.FieldInfo{
				"Name": {Type: "string", Tags: map[string]string{"json": "name"}},
				"Age":  {Type: "int", Tags: map[string]string{"json": "age"}},
			}
			fieldOrder := []string{"Name", "Age"}

			hash1 := detect.ComputeStructHash(parsed)
			hash2 := detect.ComputeStructHashFromFields("User", fields, fieldOrder)
			Expect(hash1).To(Equal(hash2))
		})
	})

	Describe("ComputeCompositeHash", func() {
		It("should be deterministic", func() {
			structHash := "abc123def456"
			deps := map[string]string{
				"pkg/a.TypeA": "v1:hash1",
				"pkg/b.TypeB": "v2:hash2",
			}

			hash1 := detect.ComputeCompositeHash(structHash, deps)
			hash2 := detect.ComputeCompositeHash(structHash, deps)
			Expect(hash1).To(Equal(hash2))
		})

		It("should differ when struct hash changes", func() {
			deps := map[string]string{"dep": "v1:hash"}

			hash1 := detect.ComputeCompositeHash("struct1", deps)
			hash2 := detect.ComputeCompositeHash("struct2", deps)
			Expect(hash1).ToNot(Equal(hash2))
		})

		It("should differ when dependency hash changes", func() {
			structHash := "same"

			hash1 := detect.ComputeCompositeHash(structHash, map[string]string{"dep": "v1:hash1"})
			hash2 := detect.ComputeCompositeHash(structHash, map[string]string{"dep": "v2:hash2"})
			Expect(hash1).ToNot(Equal(hash2))
		})

		It("should handle nil dependencies", func() {
			hash := detect.ComputeCompositeHash("struct", nil)
			Expect(hash).ToNot(BeEmpty())
		})

		It("should handle empty dependencies", func() {
			hash1 := detect.ComputeCompositeHash("struct", nil)
			hash2 := detect.ComputeCompositeHash("struct", map[string]string{})
			Expect(hash1).To(Equal(hash2))
		})
	})
})
