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

func newToken(key SessionKey) (string, []byte, error) {
	secret := make([]byte, secretLen)
	if _, err := rand.Read(secret); err != nil {
		return "", nil, err
	}
	raw := append(key[:], secret...)
	return tokenPrefix + tokenEncoding.EncodeToString(raw), hashSecret(secret), nil
}

// parseToken returns [ErrInvalidToken] for a malformed token.
func parseToken(token string) (SessionKey, []byte, error) {
	if !strings.HasPrefix(token, tokenPrefix) {
		return uuid.Nil(), nil, ErrInvalidToken
	}
	raw, err := tokenEncoding.DecodeString(strings.TrimPrefix(token, tokenPrefix))
	if err != nil || len(raw) != 16+secretLen {
		return uuid.Nil(), nil, ErrInvalidToken
	}
	var key SessionKey
	copy(key[:], raw[:16])
	return key, hashSecret(raw[16:]), nil
}

func hashSecret(secret []byte) []byte {
	sum := sha256.Sum256(secret)
	return sum[:]
}
