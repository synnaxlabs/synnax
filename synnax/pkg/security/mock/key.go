// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"crypto"
	"crypto/rsa"
)

// KeyProvider is a mock implementation of security.KeyProvider
// that wraps an RSA private key.
type KeyProvider struct{ Key *rsa.PrivateKey }

// NodePrivate implements security.KeyProvider.
func (m KeyProvider) NodePrivate() crypto.PrivateKey { return m.Key }
