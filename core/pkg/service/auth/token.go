// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"uuid"
)

const tokenPrefix = "sys_"

const secretLen = 32

var tokenEncoding = base64.RawURLEncoding

// NewToken returns a token made of prefix, key, and a random secret, and the SHA-256
// hash of the secret to store in place of the token.
func NewToken(prefix string, key uuid.UUID) (token string, hash []byte, err error) {
	secret := make([]byte, secretLen)
	if _, err = rand.Read(secret); err != nil {
		return "", nil, err
	}
	raw := append(key[:], secret...)
	return prefix + tokenEncoding.EncodeToString(raw), hashSecret(secret), nil
}

// ParseToken returns the key and secret hash of a token made by [NewToken] with the
// same prefix. It returns [ErrInvalidToken] for a malformed token.
func ParseToken(prefix, token string) (key uuid.UUID, hash []byte, err error) {
	if !strings.HasPrefix(token, prefix) {
		return uuid.Nil(), nil, ErrInvalidToken
	}
	raw, err := tokenEncoding.DecodeString(strings.TrimPrefix(token, prefix))
	if err != nil || len(raw) != 16+secretLen {
		return uuid.Nil(), nil, ErrInvalidToken
	}
	copy(key[:], raw[:16])
	return key, hashSecret(raw[16:]), nil
}

func hashSecret(secret []byte) []byte {
	sum := sha256.Sum256(secret)
	return sum[:]
}
