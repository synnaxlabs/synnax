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

var (
	ErrInvalid  = errors.New(base64.MustDecode("aW52YWxpZCBsaWNlbnNlIGtleQ=="))
	formatRegex = regexp.MustCompile(`^\d{6}-\d{8}-\d{10}$`)
)

func parseLicenseKey(licenseKey string) (info, error) {
	if !formatRegex.MatchString(licenseKey) {
		return info{}, ErrInvalid
	}
	parts := strings.Split(licenseKey, "-")
	var (
		i   info
		err error
	)
	if i.exprTime, err = decodeDate(parts[0]); err != nil {
		return info{}, err
	}
	if i.numCh, err = getChannelCount(parts[1]); err != nil {
		return info{}, err
	}
	if err = validateChecksum(parts[2]); err != nil {
		return info{}, err
	}
	return i, nil
}

func decodeDate(datePart string) (time.Time, error) {
	year, err := strconv.Atoi(datePart[:2])
	if err != nil {
		return time.Time{}, err
	}
	year, err = crypto.Cipher(year, 89, 2)
	if err != nil {
		return time.Time{}, err
	}
	month, err := strconv.Atoi(datePart[2:4])
	if err != nil {
		return time.Time{}, err
	}
	month, err = crypto.Cipher(month, 43, 2)
	if err != nil {
		return time.Time{}, err
	}
	day, err := strconv.Atoi(datePart[4:6])
	if err != nil {
		return time.Time{}, err
	}
	day, err = crypto.Cipher(day, 77, 2)
	if err != nil {
		return time.Time{}, err
	}
	year += 2000
	if !date.DateExists(year, month, day) {
		return time.Time{}, ErrInvalid
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), nil
}

func getChannelCount(channelPart string) (types.Uint20, error) {
	numChannels, err := strconv.Atoi(channelPart)
	if err != nil {
		return 0, err
	}
	numChannels, err = crypto.Cipher(numChannels, 64317284, 8)
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
			return ErrInvalid
		}
		digits[i] = d
	}
	firstFive, err := strconv.Atoi(code[:5])
	if err != nil {
		return ErrInvalid
	}
	sum := digits[5] + digits[6] + digits[7] + digits[8]
	switch {
	case digits[1] != 4,
		firstFive%9 != 0,
		sum%7 != 0,
		digits[9] < 3,
		digits[9] > 6:
		return ErrInvalid
	}
	return nil
}
