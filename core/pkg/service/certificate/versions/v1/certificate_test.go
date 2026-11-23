// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/certificate/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	secret "github.com/synnaxlabs/synnax/pkg/service/secret/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Certificate", func() {
	var (
		key     uuid.UUID
		details v1.Details
		stored  v1.Certificate
		file    v1.Certificate
	)
	BeforeEach(func() {
		key = uuid.New()
		details = v1.Details{Subject: "CN=plant", NotAfter: 10}
		stored = v1.Certificate{Variant: v1.StoredCertificate{
			Base:    v1.Base{Key: key, Name: "Plant"},
			Details: details,
		}}
		file = v1.Certificate{Variant: v1.FileCertificate{
			Base:            v1.Base{Key: key, Name: "Plant file"},
			CertificatePath: "/c.pem",
		}}
	})

	Describe("Source", func() {
		It("Should return the source of each variant", func() {
			Expect(stored.Source()).To(Equal(secret.SourceStored))
			Expect(file.Source()).To(Equal(secret.SourceFile))
		})
		It("Should return an empty source for a certificate with no variant", func() {
			Expect(v1.Certificate{}.Source()).To(BeEmpty())
		})
	})

	Describe("Details", func() {
		It("Should return the details of a stored certificate", func() {
			Expect(MustBeOk(stored.Details())).To(Equal(details))
		})
		It("Should return the details of a file certificate that has them", func() {
			variant, ok := file.Variant.(v1.FileCertificate)
			Expect(ok).To(BeTrue())
			variant.Details = &details
			Expect(MustBeOk(v1.Certificate{Variant: variant}.Details())).
				To(Equal(details))
		})
		It("Should report no details for a file certificate without them", func() {
			Expect(file.Details()).To(BeZero())
		})
		It("Should report no details for a certificate with no variant", func() {
			Expect(v1.Certificate{}.Details()).To(BeZero())
		})
	})

	Describe("GorpKey", func() {
		It("Should return the key of the certificate", func() {
			Expect(stored.GorpKey()).To(Equal(key))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(stored.SetOptions()).To(BeNil())
		})
	})

	Describe("ResourceID", func() {
		It("Should return a certificate resource ID with the key", func() {
			Expect(stored.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeCertificate,
				Key:  key.String(),
			}))
		})
	})
})
