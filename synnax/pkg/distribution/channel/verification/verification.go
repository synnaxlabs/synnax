// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package verification

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/synnaxlabs/x/crypto"
	"github.com/synnaxlabs/x/date"
	"github.com/synnaxlabs/x/types"
)

var errInvalidInput error

const (
	freeCount   = 50
	yearCipher  = 89
	monthCipher = 43
	dayCipher   = 77
)

func checkFormat(input string) error {
	inputLength := 26
	numDashes := 2
	numParts := 3
	firstPartLength := 6
	secondPartLength := 8
	thirdPartLength := 10
	errFormat := errors.New(decode("cHJvZHVjdCBsaWNlbnNlIGtleSBpcyBpbiBhbiBpbnZhbGlkIGZvcm1hdA=="))
	if len(input) != inputLength {
		return errFormat
	}
	dashCount := strings.Count(input, "-")
	if dashCount != numDashes {
		return errFormat
	}
	parts := strings.Split(input, "-")
	if (len(parts) != numParts) || (len(parts[0]) != firstPartLength) ||
		(len(parts[1]) != secondPartLength) || (len(parts[2]) != thirdPartLength) {
		return errFormat
	}
	_, err := strconv.Atoi(parts[0])
	if err != nil {
		return errFormat
	}
	_, err = strconv.Atoi(parts[1])
	if err != nil {
		return errFormat
	}
	_, err = strconv.Atoi(parts[2])
	if err != nil {
		return errFormat
	}
	return nil
}

func validateInput(input string) error {
	err := checkFormat(input)
	if err != nil {
		return err
	}
	parts := strings.Split(input, "-")
	year, _ := strconv.Atoi(parts[0][0:2])
	month, _ := strconv.Atoi(parts[0][2:4])
	day, _ := strconv.Atoi(parts[0][4:6])
	year, err = crypto.Cipher(year, yearCipher, 2)
	if err != nil {
		return errInvalidInput
	}
	year += 2000
	month, err = crypto.Cipher(month, monthCipher, 2)
	if err != nil {
		return errInvalidInput
	}
	day, err = crypto.Cipher(day, dayCipher, 2)
	if err != nil {
		return errInvalidInput
	}
	if !date.DateExists(year, month, day) {
		return errInvalidInput
	}
	return inputCheckFunc(input)
}

func whenStale(input string) time.Time {
	parts := strings.Split(input, "-")
	year, _ := strconv.Atoi(parts[0][0:2])
	month, _ := strconv.Atoi(parts[0][2:4])
	day, _ := strconv.Atoi(parts[0][4:6])
	year, _ = crypto.Cipher(year, yearCipher, 2)
	year += 2000
	month, _ = crypto.Cipher(month, monthCipher, 2)
	day, _ = crypto.Cipher(day, dayCipher, 2)
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
}

func getNumChan(input string) types.Uint20 {
	channelCipher := 64317284
	parts := strings.Split(input, "-")
	numChannels, _ := strconv.Atoi(parts[1])
	numChannels, _ = crypto.Cipher(numChannels, channelCipher, 8)
	return types.Uint20(numChannels)
}

func inputCheckFunc(input string) error {
	code := strings.Split(input, "-")[2]
	var digits [10]int
	for i := 0; i < 10; i++ {
		digits[i], _ = strconv.Atoi(string(code[i]))
	}
	firstFive, _ := strconv.Atoi(code[0:5])
	sum := digits[5] + digits[6] + digits[7] + digits[8]
	if digits[1] != 4 {
		return errInvalidInput
	} else if firstFive%9 != 0 {
		return errInvalidInput
	} else if sum%7 != 0 {
		return errInvalidInput
	} else if digits[9] > 6 || digits[9] < 3 {
		return errInvalidInput
	}
	return nil
}

func init() {
	errInvalidInput = errors.New(decode("aW52YWxpZCBwcm9kdWN0IGxpY2Vuc2Uga2V5"))
}

func decode(str string) string {
	msg, _ := base64.StdEncoding.DecodeString(str)
	return string(msg)
}
