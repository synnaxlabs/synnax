// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package license

import (
	"crypto/sha256"
	"regexp"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/crypto"
	"github.com/synnaxlabs/x/date"
	"github.com/synnaxlabs/x/errors"
)

// legacyFormat matches a key in the unsigned numeric format: an expiry date, a channel
// count, and a checksum.
var legacyFormat = regexp.MustCompile(`^\d{6}-\d{8}-\d{10}$`)

// ParseLegacy returns the license a key in the unsigned numeric format carries: an
// enterprise license that expires at the start of the key's date in local time, caps
// channels at the key's count, and runs on any machine. Its Jti is derived from key, so
// the same key always gives the same license. Returns ErrInvalid on a malformed key, a
// bad checksum, a date that does not exist, or a channel count of zero.
func ParseLegacy(key string) (License, error) {
	if !legacyFormat.MatchString(key) {
		return License{}, errors.Wrap(ErrInvalid, "not a numeric license key")
	}
	parts := strings.Split(key, "-")
	year := 2000 + unshift(parts[0][:2], 89)
	month, day := unshift(parts[0][2:4], 43), unshift(parts[0][4:6], 77)
	if !date.DateExists(year, month, day) {
		return License{}, errors.Wrap(ErrInvalid, "bad expiry date")
	}
	channels := unshift(parts[1], 64317284)
	if channels == 0 {
		return License{}, errors.Wrap(ErrInvalid, "the license key covers no channels")
	}
	if !legacyChecksumValid(parts[2]) {
		return License{}, errors.Wrap(ErrInvalid, "bad checksum")
	}
	exp := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	sum := sha256.Sum256([]byte(key))
	jti := uuid.UUID(sum[:16])
	jti[6] = jti[6]&0x0f | 0x80
	jti[8] = jti[8]&0x3f | 0x80
	return License{
		Jti:           jti,
		Exp:           new(uint32(exp.Unix())),
		ClaimsVersion: claimsVersion,
		Edition:       EditionEnterprise,
		Fingerprints:  []string{},
		Channels:      uint32(channels),
		Required:      []string{},
	}, nil
}

// unshift reverses the digit shift a numeric key applies to each of its fields.
func unshift(digits string, distance int) int {
	return lo.Must(crypto.Cipher(lo.Must(strconv.Atoi(digits)), distance, len(digits)))
}

func legacyChecksumValid(sum string) bool {
	digit := func(i int) int { return int(sum[i] - '0') }
	return digit(1) == 4 &&
		lo.Must(strconv.Atoi(sum[:5]))%9 == 0 &&
		(digit(5)+digit(6)+digit(7)+digit(8))%7 == 0 &&
		digit(9) >= 3 &&
		digit(9) <= 6
}
