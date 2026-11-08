// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate

import (
	"encoding/json/v2"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Resources returns the resource.Provider of certificates.
func (s *Service) Resources() *resource.TableProvider[Key, Certificate] {
	return resource.NewTableProvider(resource.TableProviderConfig[Key, Certificate]{
		Table:  s.table,
		Search: s.indexes.search,
		ID:     ResourceID,
		Key:    resource.ParseUUIDKey,
		Name:   func(c *Certificate) string { return c.Base().Name },
	})
}

// ResourceID returns the resource ID of the certificate with the given key.
func ResourceID(key Key) resource.ID {
	return resource.ID{Type: resource.TypeCertificate, Key: key.String()}
}

// ResourceIDs returns the resource IDs of the certificates with the given keys.
func ResourceIDs(keys []Key) []resource.ID {
	return lo.Map(keys, func(k Key, _ int) resource.ID { return ResourceID(k) })
}

// ResourceIDsFromCertificates returns the resource IDs of the given certificates.
func ResourceIDsFromCertificates(certs []Certificate) []resource.ID {
	return lo.Map(
		certs,
		func(c Certificate, _ int) resource.ID { return c.ResourceID() },
	)
}

// SetRecord returns the resource ID of the certificate that a sample of the set channel
// carries.
func SetRecord(sample []byte) (resource.ID, error) {
	var c Certificate
	if err := json.Unmarshal(sample, &c); err != nil {
		return resource.ID{}, err
	}
	return c.ResourceID(), nil
}

// DeleteRecord returns the resource ID of the certificate that a sample of the delete
// channel names. It returns a validation error when the sample is not a key.
func DeleteRecord(sample []byte) (resource.ID, error) {
	if len(sample) != len(Key{}) {
		return resource.ID{}, errors.Wrapf(
			validate.ErrValidation,
			"certificate delete sample is %d bytes, not %d",
			len(sample),
			len(Key{}),
		)
	}
	return ResourceID(Key(sample)), nil
}
