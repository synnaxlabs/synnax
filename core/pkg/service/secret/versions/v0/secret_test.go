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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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
			Variant: v0.FileSecret{Base: v0.Base{Key: key, Name: "Key"}, FileName: "k"},
		}
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

	Describe("ResourceID", func() {
		It("Should return a secret resource ID with the key", func() {
			Expect(stored.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeSecret,
				Key:  key.String(),
			}))
		})
	})
})
