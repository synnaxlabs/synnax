// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// EncryptionKey is the AES-256 key that encrypts the value of every stored secret.
type EncryptionKey [32]byte

// NewEncryptionKey returns a random encryption key.
func NewEncryptionKey() EncryptionKey {
	var k EncryptionKey
	lo.Must(rand.Read(k[:]))
	return k
}

// ParseEncryptionKey decodes an encryption key from standard base64. It returns a
// validation error unless s decodes to exactly 32 bytes.
func ParseEncryptionKey(s string) (EncryptionKey, error) {
	var k EncryptionKey
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return k, errors.Wrap(validate.ErrValidation, "secrets key is not base64")
	}
	if len(b) != len(k) {
		return k, errors.Wrapf(
			validate.ErrValidation,
			"secrets key is %d bytes, not %d",
			len(b),
			len(k),
		)
	}
	copy(k[:], b)
	return k, nil
}

// LoadEncryptionKey reads the base64 encryption key in the file at path. When no file
// exists there, it writes a new random key to path, readable only by its owner, and
// returns it.
func LoadEncryptionKey(path string) (EncryptionKey, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		k := NewEncryptionKey()
		encoded := base64.StdEncoding.EncodeToString(k[:])
		if err = os.WriteFile(path, []byte(encoded), xfs.UserRW); err != nil {
			return EncryptionKey{}, err
		}
		return k, nil
	}
	if err != nil {
		return EncryptionKey{}, err
	}
	k, err := ParseEncryptionKey(string(b))
	if err != nil {
		return EncryptionKey{}, errors.Wrapf(err, "invalid secrets key in %s", path)
	}
	return k, nil
}

const grantKeyInfo = "synnax secret grant"

var (
	checkKey       = []byte("sy_secret_encryption_check")
	checkPlaintext = []byte("synnax")
)

// openCipher returns the AEAD that encrypts secret values under k and the key that
// signs grants. It returns an error when k cannot decrypt the check value that an
// earlier start stored in db, and stores the check value when there is none.
func openCipher(
	ctx context.Context,
	db *gorp.DB,
	k EncryptionKey,
) (cipher.AEAD, []byte, error) {
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, nil, err
	}
	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, nil, err
	}
	grantKey, err := hkdf.Key(sha256.New, k[:], nil, grantKeyInfo, sha256.Size)
	if err != nil {
		return nil, nil, err
	}
	stored, closer, err := db.Get(ctx, checkKey)
	if errors.Is(err, query.ErrNotFound) {
		check := aead.Seal(nil, nil, checkPlaintext, checkKey)
		return aead, grantKey, db.Set(ctx, checkKey, check)
	}
	if err != nil {
		return nil, nil, err
	}
	_, openErr := aead.Open(nil, nil, stored, checkKey)
	if err = closer.Close(); err != nil {
		return nil, nil, err
	}
	if openErr != nil {
		return nil, nil, errors.New(
			"the secrets key does not match the key that encrypted this Core's secrets",
		)
	}
	return aead, grantKey, nil
}
