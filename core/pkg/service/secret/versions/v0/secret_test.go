// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/secret/versions/v0"
)

var _ = Describe("Secret", func() {
	var (
		key    uuid.UUID
		stored v0.Secret
		file   v0.Secret
	)
	BeforeEach(func() {
		key = uuid.New()
		stored = v0.Secret{
			Variant: v0.StoredSecret{Base: v0.Base{Key: key, Name: "Password"}},
		}
		file = v0.Secret{
			Variant: v0.FileSecret{Base: v0.Base{Key: key, Name: "Key"}, Path: "/k"},
		}
	})

	Describe("Base", func() {
		It("Should return the base of each variant", func() {
			Expect(stored.Base()).To(Equal(v0.Base{Key: key, Name: "Password"}))
			Expect(file.Base()).To(Equal(v0.Base{Key: key, Name: "Key"}))
		})
		It("Should return a zero base for a secret with no variant", func() {
			Expect(v0.Secret{}.Base()).To(Equal(v0.Base{}))
		})
	})

	Describe("Source", func() {
		It("Should return the source of each variant", func() {
			Expect(stored.Source()).To(Equal(v0.SourceStored))
			Expect(file.Source()).To(Equal(v0.SourceFile))
		})
		It("Should return an empty source for a secret with no variant", func() {
			Expect(v0.Secret{}.Source()).To(BeEmpty())
		})
	})

	Describe("WithBase", func() {
		It("Should replace the base and keep the variant fields", func() {
			base := v0.Base{Key: uuid.New(), Name: "Renamed", Updated: 5}
			Expect(file.WithBase(base)).To(Equal(v0.Secret{
				Variant: v0.FileSecret{Base: base, Path: "/k"},
			}))
			Expect(stored.WithBase(base).Base()).To(Equal(base))
		})
		It("Should leave a secret with no variant unchanged", func() {
			Expect(v0.Secret{}.WithBase(v0.Base{Name: "A"})).To(Equal(v0.Secret{}))
		})
	})

	Describe("GorpKey", func() {
		It("Should return the key of the secret", func() {
			Expect(stored.GorpKey()).To(Equal(key))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(stored.SetOptions()).To(BeNil())
		})
	})

	Describe("OntologyID", func() {
		It("Should return a secret ontology ID with the key", func() {
			Expect(stored.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeSecret,
				Key:  key.String(),
			}))
		})
	})
})
