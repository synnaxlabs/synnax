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
	v0 "github.com/synnaxlabs/synnax/pkg/service/certificate/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	secret "github.com/synnaxlabs/synnax/pkg/service/secret/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Certificate", func() {
	var (
		key     uuid.UUID
		details v0.Details
		stored  v0.Certificate
		file    v0.Certificate
	)
	BeforeEach(func() {
		key = uuid.New()
		details = v0.Details{Subject: "CN=plant", NotAfter: 10}
		stored = v0.Certificate{Variant: v0.StoredCertificate{
			Base:    v0.Base{Key: key, Name: "Plant"},
			Details: details,
		}}
		file = v0.Certificate{Variant: v0.FileCertificate{
			Base:            v0.Base{Key: key, Name: "Plant file"},
			CertificatePath: "/c.pem",
		}}
	})

	Describe("Base", func() {
		It("Should return the base of each variant", func() {
			Expect(stored.Base()).To(Equal(v0.Base{Key: key, Name: "Plant"}))
			Expect(file.Base()).To(Equal(v0.Base{Key: key, Name: "Plant file"}))
		})
		It("Should return a zero base for a certificate with no variant", func() {
			Expect(v0.Certificate{}.Base()).To(Equal(v0.Base{}))
		})
	})

	Describe("Source", func() {
		It("Should return the source of each variant", func() {
			Expect(stored.Source()).To(Equal(secret.SourceStored))
			Expect(file.Source()).To(Equal(secret.SourceFile))
		})
		It("Should return an empty source for a certificate with no variant", func() {
			Expect(v0.Certificate{}.Source()).To(BeEmpty())
		})
	})

	Describe("Details", func() {
		It("Should return the details of a stored certificate", func() {
			Expect(MustBeOk(stored.Details())).To(Equal(details))
		})
		It("Should return the details of a file certificate that has them", func() {
			variant := file.Variant.(v0.FileCertificate)
			variant.Details = &details
			Expect(MustBeOk(v0.Certificate{Variant: variant}.Details())).
				To(Equal(details))
		})
		It("Should report no details for a file certificate without them", func() {
			Expect(file.Details()).To(BeZero())
		})
		It("Should report no details for a certificate with no variant", func() {
			Expect(v0.Certificate{}.Details()).To(BeZero())
		})
	})

	Describe("WithBase", func() {
		It("Should replace the base and keep the variant fields", func() {
			base := v0.Base{Key: uuid.New(), Name: "Renamed"}
			Expect(file.WithBase(base)).To(Equal(v0.Certificate{
				Variant: v0.FileCertificate{Base: base, CertificatePath: "/c.pem"},
			}))
			Expect(stored.WithBase(base).Base()).To(Equal(base))
		})
		It("Should leave a certificate with no variant unchanged", func() {
			Expect(v0.Certificate{}.WithBase(v0.Base{Name: "A"})).
				To(Equal(v0.Certificate{}))
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

	Describe("OntologyID", func() {
		It("Should return a certificate ontology ID with the key", func() {
			Expect(stored.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeCertificate,
				Key:  key.String(),
			}))
		})
	})
})
