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
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// EncryptionKey is a 32-byte AES-256 key.
type EncryptionKey [32]byte

// NewEncryptionKey returns a random encryption key.
func NewEncryptionKey() EncryptionKey {
	var k EncryptionKey
	lo.Must(rand.Read(k[:]))
	return k
}

// ParseEncryptionKey decodes an encryption key from standard base64. It returns a
// validation error unless s decodes to exactly 32 bytes that are not all zeros.
func ParseEncryptionKey(s string) (EncryptionKey, error) {
	var k EncryptionKey
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return EncryptionKey{}, errors.Wrap(
			validate.ErrValidation,
			"secrets key is not base64",
		)
	}
	if len(b) != len(k) {
		return EncryptionKey{}, errors.Wrapf(
			validate.ErrValidation,
			"secrets key is %d bytes, not %d",
			len(b),
			len(k),
		)
	}
	copy(k[:], b)
	if k == (EncryptionKey{}) {
		return EncryptionKey{}, errors.Wrap(
			validate.ErrValidation,
			"secrets key is all zeros",
		)
	}
	return k, nil
}

// LoadEncryptionKey reads the base64 encryption key in the file at path. When no file
// exists there, it writes a new random key to path, readable only by its owner, and
// returns it. When another process creates the file first, it returns the key in that
// file.
func LoadEncryptionKey(path string) (EncryptionKey, error) {
	k, err := readEncryptionKey(path)
	if !errors.Is(err, os.ErrNotExist) {
		return k, err
	}
	k = NewEncryptionKey()
	err = createExclusive(path, base64.StdEncoding.EncodeToString(k[:]))
	if errors.Is(err, os.ErrExist) {
		return readEncryptionKey(path)
	}
	if err != nil {
		return EncryptionKey{}, err
	}
	return k, nil
}

func readEncryptionKey(path string) (EncryptionKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return EncryptionKey{}, err
	}
	k, err := ParseEncryptionKey(string(b))
	if err != nil {
		return EncryptionKey{}, errors.Wrapf(err, "invalid secrets key in %s", path)
	}
	return k, nil
}

// createExclusive creates a file at path that holds data and that only its owner can
// read. It returns an error that wraps os.ErrExist when the file exists. The file
// appears with all of data, so a reader never sees it partly written, and it is on disk
// when createExclusive returns.
func createExclusive(path, data string) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, os.Remove(tmp.Name())) }()
	_, err = tmp.WriteString(data)
	err = errors.Combine(err, tmp.Sync())
	if err = errors.Combine(err, tmp.Close()); err != nil {
		return err
	}
	if err = os.Link(tmp.Name(), path); err != nil {
		return err
	}
	return syncDir(dir)
}

var dataKeyKey = []byte("sy_secret_data_key")

// openCipher returns the AEAD for secret values, under a data key that db stores
// encrypted under k. The first call creates the data key. It returns an error when k
// cannot decrypt the stored data key.
func openCipher(
	ctx context.Context,
	db *gorp.DB,
	k EncryptionKey,
) (cipher.AEAD, error) {
	outer, err := newAEAD(k)
	if err != nil {
		return nil, err
	}
	dataKey, err := loadDataKey(ctx, db, outer)
	if err != nil {
		return nil, err
	}
	return newAEAD(dataKey)
}

func newAEAD(k EncryptionKey) (cipher.AEAD, error) {
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCMWithRandomNonce(block)
}

func loadDataKey(
	ctx context.Context,
	db *gorp.DB,
	outer cipher.AEAD,
) (EncryptionKey, error) {
	sealed, closer, err := db.Get(ctx, dataKeyKey)
	if errors.Is(err, query.ErrNotFound) {
		dataKey := NewEncryptionKey()
		sealed = outer.Seal(nil, nil, dataKey[:], dataKeyKey)
		if err = db.Set(ctx, dataKeyKey, sealed); err != nil {
			return EncryptionKey{}, err
		}
		return dataKey, nil
	}
	if err != nil {
		return EncryptionKey{}, err
	}
	plain, openErr := outer.Open(nil, nil, sealed, dataKeyKey)
	if err = closer.Close(); err != nil {
		return EncryptionKey{}, err
	}
	if openErr != nil {
		return EncryptionKey{}, errors.New(
			"the secrets key does not match the key that encrypted this Core's secrets",
		)
	}
	return EncryptionKey(plain), nil
}
