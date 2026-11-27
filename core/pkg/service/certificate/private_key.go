// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate

// HasPrivateKey reports whether c holds a private key or names a private key file.
func (c Certificate) HasPrivateKey() bool {
	switch v := c.Variant.(type) {
	case StoredCertificate:
		return v.PrivateKey != nil
	case FileCertificate:
		return v.PrivateKeyFile != nil
	default:
		return false
	}
}
