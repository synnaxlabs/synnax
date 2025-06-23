// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/synnaxlabs/x/crypto"
	"github.com/synnaxlabs/x/date"
	"github.com/synnaxlabs/x/encoding/base64"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
)

type info struct {
	numCh    types.Uint20
	exprTime time.Time
}

const (
	yearCipher    = 89
	monthCipher   = 43
	dayCipher     = 77
	channelCipher = 64317284
)

var (
	errInvalid  = errors.New(base64.MustDecode("aW52YWxpZCBsaWNlbnNlIGtleQ=="))
	formatRegex = regexp.MustCompile(`^\d{6}-\d{8}-\d{10}$`)
)

func parseLicenseKey(licenseKey string) (info, error) {
	if !formatRegex.MatchString(licenseKey) {
		return info{}, errInvalid
	}
	parts := strings.Split(licenseKey, "-")
	expirationTime, err := decodeDate(parts[0])
	if err != nil {
		return info{}, err
	}
	numChannels, err := getChannelCount(parts[1])
	if err != nil {
		return info{}, err
	}
	if err = validateChecksum(parts[2]); err != nil {
		return info{}, err
	}
	return info{numCh: numChannels, exprTime: expirationTime}, nil
}

func decodeDate(datePart string) (time.Time, error) {
	var err error
	var year, month, day int
	if year, err = strconv.Atoi(datePart[:2]); err != nil {
		return time.Time{}, err
	}
	if month, err = strconv.Atoi(datePart[2:4]); err != nil {
		return time.Time{}, err
	}
	if day, err = strconv.Atoi(datePart[4:6]); err != nil {
		return time.Time{}, err
	}
	if year, err = crypto.Cipher(year, yearCipher, 2); err != nil {
		return time.Time{}, err
	}
	if month, err = crypto.Cipher(month, monthCipher, 2); err != nil {
		return time.Time{}, err
	}
	if day, err = crypto.Cipher(day, dayCipher, 2); err != nil {
		return time.Time{}, err
	}
	year += 2000
	if !date.DateExists(year, month, day) {
		return time.Time{}, errInvalid
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

func getChannelCount(channelPart string) (types.Uint20, error) {
	numChannels, err := strconv.Atoi(channelPart)
	if err != nil {
		return 0, err
	}
	numChannels, err = crypto.Cipher(numChannels, channelCipher, 8)
	if err != nil {
		return 0, err
	}
	return types.Uint20(numChannels), nil
}

func validateChecksum(code string) error {
	var digits [10]int
	for i := range digits {
		d, err := strconv.Atoi(string(code[i]))
		if err != nil {
			return errInvalid
		}
		digits[i] = d
	}
	firstFive, err := strconv.Atoi(code[:5])
	if err != nil {
		return errInvalid
	}
	sum := digits[5] + digits[6] + digits[7] + digits[8]
	switch {
	case digits[1] != 4,
		firstFive%9 != 0,
		sum%7 != 0,
		digits[9] < 3,
		digits[9] > 6:
		return errInvalid
	}
	return nil
}
