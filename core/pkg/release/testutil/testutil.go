// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil signs files as the Synnax release workflow does.
package testutil

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"aead.dev/minisign"
	"github.com/samber/lo"
)

// Signer signs files with its own key.
type Signer struct {
	// PublicKey is the minisign public key that checks the signatures of the Signer.
	PublicKey string
	key       minisign.PrivateKey
}

// NewSigner returns a Signer with a new key.
func NewSigner() Signer {
	pub, key := lo.Must2(minisign.GenerateKey(rand.Reader))
	return Signer{PublicKey: pub.String(), key: key}
}

// Sign returns the signature file of data in the file name, in the format of the Tauri
// signer.
func (s Signer) Sign(name string, data []byte) []byte {
	r := minisign.NewReader(bytes.NewReader(data))
	lo.Must(io.Copy(io.Discard, r))
	trusted := fmt.Sprintf("timestamp:%d\tfile:%s", time.Now().Unix(), name)
	signature := r.SignWithComments(s.key, trusted, "signature from tauri secret key")
	return []byte(base64.StdEncoding.EncodeToString(signature))
}
