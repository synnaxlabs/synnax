// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package password

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/errors"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// errMalformedHash is returned when a stored hash does not parse as its tag claims.
var errMalformedHash = errors.New("malformed password hash")

// verifier checks the hashes of one algorithm. Hashes carry a tag naming the algorithm,
// so a verifier can tell which stored hashes are its own.
type verifier interface {
	// verify returns nil when plaintext matches hash, [auth.ErrInvalidCredentials]
	// when it does not, and another error when hash is malformed.
	verify(hash []byte, plaintext string) error
	// made reports whether this algorithm produced hash.
	made(hash []byte) bool
}

// hasher is a verifier that also makes new hashes.
type hasher interface {
	verifier
	// hash returns a new tagged hash of plaintext.
	hash(plaintext string) ([]byte, error)
}

const (
	saltLen = 16
	keyLen  = 32
)

func salt() ([]byte, error) {
	s := make([]byte, saltLen)
	_, err := rand.Read(s)
	return s, err
}

var b64 = base64.RawStdEncoding

// phc splits a PHC-formatted hash, "$id$params$salt$hash", into its parts.
func phc(hash []byte) (id, params string, salt, key []byte, err error) {
	parts := strings.Split(string(hash), "$")
	if len(parts) == 5 {
		parts = append(parts[:2], append([]string{""}, parts[2:]...)...)
	}
	if len(parts) != 6 {
		return "", "", nil, nil, errMalformedHash
	}
	if salt, err = b64.DecodeString(parts[4]); err != nil {
		return "", "", nil, nil, errors.Combine(errMalformedHash, err)
	}
	if key, err = b64.DecodeString(parts[5]); err != nil {
		return "", "", nil, nil, errors.Combine(errMalformedHash, err)
	}
	return parts[1], parts[3], salt, key, nil
}

func compare(stored, computed []byte) error {
	if subtle.ConstantTimeCompare(stored, computed) != 1 {
		return auth.ErrInvalidCredentials
	}
	return nil
}

// argon2id hashes with Argon2id at the parameters OWASP recommends.
type argon2id struct{}

const (
	argon2idTag     = "$argon2id$"
	argon2idMemory  = 19 * 1024
	argon2idTime    = 2
	argon2idThreads = 1
)

func (argon2id) hash(plaintext string) ([]byte, error) {
	s, err := salt()
	if err != nil {
		return nil, err
	}
	key := argon2.IDKey(
		[]byte(plaintext),
		s,
		argon2idTime,
		argon2idMemory,
		argon2idThreads,
		keyLen,
	)
	return fmt.Appendf(nil,
		"%sv=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2idTag,
		argon2.Version,
		argon2idMemory,
		argon2idTime,
		argon2idThreads,
		b64.EncodeToString(s),
		b64.EncodeToString(key),
	), nil
}

func (argon2id) verify(hash []byte, plaintext string) error {
	_, params, s, key, err := phc(hash)
	if err != nil {
		return err
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(
		params,
		"m=%d,t=%d,p=%d",
		&memory,
		&time,
		&threads,
	); err != nil {
		return errors.Combine(errMalformedHash, err)
	}
	computed := argon2.IDKey(
		[]byte(plaintext),
		s,
		time,
		memory,
		threads,
		uint32(len(key)),
	)
	return compare(key, computed)
}

func (argon2id) made(hash []byte) bool {
	return bytes.HasPrefix(hash, []byte(argon2idTag))
}

// pbkdf2SHA256 hashes with PBKDF2-HMAC-SHA256, the one password hash NIST approves,
// at the iteration count OWASP recommends.
type pbkdf2SHA256 struct{}

const (
	pbkdf2Tag        = "$pbkdf2-sha256$"
	pbkdf2Iterations = 600_000
)

func (pbkdf2SHA256) hash(plaintext string) ([]byte, error) {
	s, err := salt()
	if err != nil {
		return nil, err
	}
	key, err := pbkdf2.Key(sha256.New, plaintext, s, pbkdf2Iterations, keyLen)
	if err != nil {
		return nil, err
	}
	return fmt.Appendf(nil,
		"%si=%d$%s$%s",
		pbkdf2Tag,
		pbkdf2Iterations,
		b64.EncodeToString(s),
		b64.EncodeToString(key),
	), nil
}

func (pbkdf2SHA256) verify(hash []byte, plaintext string) error {
	_, params, s, key, err := phc(hash)
	if err != nil {
		return err
	}
	var iterations int
	if _, err := fmt.Sscanf(params, "i=%d", &iterations); err != nil {
		return errors.Combine(errMalformedHash, err)
	}
	computed, err := pbkdf2.Key(sha256.New, plaintext, s, iterations, len(key))
	if err != nil {
		return errors.Combine(errMalformedHash, err)
	}
	return compare(key, computed)
}

func (pbkdf2SHA256) made(hash []byte) bool {
	return bytes.HasPrefix(hash, []byte(pbkdf2Tag))
}

// bcryptVerifier checks the bcrypt hashes stored before this package existed. It never
// makes a new one.
type bcryptVerifier struct{}

func (bcryptVerifier) verify(hash []byte, plaintext string) error {
	err := bcrypt.CompareHashAndPassword(hash, []byte(plaintext))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return auth.ErrInvalidCredentials
	}
	if err != nil {
		return errors.Combine(errMalformedHash, err)
	}
	return nil
}

func (bcryptVerifier) made(hash []byte) bool {
	return bytes.HasPrefix(hash, []byte("$2"))
}

// hashers returns the hasher for new passwords and every verifier whose hashes may be
// checked. With fips on, only PBKDF2 is allowed in either role.
func hashers(fips bool) (current hasher, accepted []verifier) {
	if fips {
		return pbkdf2SHA256{}, []verifier{pbkdf2SHA256{}}
	}
	return argon2id{}, []verifier{argon2id{}, bcryptVerifier{}, pbkdf2SHA256{}}
}
