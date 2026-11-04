// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	secret "github.com/synnaxlabs/synnax/pkg/service/secret/versions/v0"
)

var _ gorp.Entry[Key] = Certificate{}

// Base returns the fields shared by every variant of the certificate. A certificate
// with no variant has a zero Base.
func (c Certificate) Base() Base {
	switch variant := c.Variant.(type) {
	case StoredCertificate:
		return variant.Base
	case FileCertificate:
		return variant.Base
	}
	return Base{}
}

// Source returns where the contents of the certificate live, or an empty source if the
// certificate has no variant.
func (c Certificate) Source() secret.Source {
	switch c.Variant.(type) {
	case StoredCertificate:
		return secret.SourceStored
	case FileCertificate:
		return secret.SourceFile
	}
	return ""
}

// Details returns the details of the certificate's leaf, and false when they are
// unknown: a file certificate saved without details, or a certificate with no variant.
func (c Certificate) Details() (Details, bool) {
	switch variant := c.Variant.(type) {
	case StoredCertificate:
		return variant.Details, true
	case FileCertificate:
		if variant.Details != nil {
			return *variant.Details, true
		}
	}
	return Details{}, false
}

// WithBase returns a copy of the certificate with its shared fields set to base.
func (c Certificate) WithBase(base Base) Certificate {
	switch variant := c.Variant.(type) {
	case StoredCertificate:
		variant.Base = base
		c.Variant = variant
	case FileCertificate:
		variant.Base = base
		c.Variant = variant
	}
	return c
}

// GorpKey returns the certificate's storage key.
func (c Certificate) GorpKey() Key { return c.Base().Key }

// SetOptions returns no additional storage options.
func (Certificate) SetOptions() []any { return nil }

// OntologyID returns the certificate's ontology identifier.
func (c Certificate) OntologyID() ontology.ID {
	return ontology.ID{
		Type: ontology.ResourceTypeCertificate,
		Key:  c.Base().Key.String(),
	}
}
