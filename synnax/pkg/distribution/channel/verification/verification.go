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
)

const maxFreeChannels = 50
const yearCipher = 89
const monthCipher = 43
const dayCipher = 77

// checkKeyFormat checks that inputKey is in the valid format of
// ######-#####-########## (6#-5#-10#). If it is not in a correct format, it
// returns an error.
func checkKeyFormat(inputKey string) error {

	inputKeyLength := 26
	numDashes := 2
	numPartsOfKey := 3
	firstPartLength := 6
	secondPartLength := 8
	thirdPartLength := 10

	encoded := "cHJvZHVjdCBsaWNlbnNlIGtleSBpcyBpbiBhbiBpbnZhbGlkIGZvcm1hdA=="
	errKeyFormatMsg, _ := base64.StdEncoding.DecodeString(encoded)
	errKeyFormat := errors.New(string(errKeyFormatMsg))

	// right length and number of dashes
	if len(inputKey) != inputKeyLength {
		return errKeyFormat
	}
	dashCount := strings.Count(inputKey, "-")
	if dashCount != numDashes {
		return errKeyFormat
	}

	// split key, make sure it is the correct length
	parts := strings.Split(inputKey, "-")
	if len(parts) != numPartsOfKey {
		return errKeyFormat
	}

	if len(parts[0]) != firstPartLength {
		return errKeyFormat
	}
	if len(parts[1]) != secondPartLength {
		return errKeyFormat
	}
	if len(parts[2]) != thirdPartLength {
		return errKeyFormat
	}

	// make sure key is all integers
	_, err := strconv.Atoi(parts[0])
	if err != nil {
		return errKeyFormat
	}
	_, err = strconv.Atoi(parts[1])
	if err != nil {
		return errKeyFormat
	}
	_, err = strconv.Atoi(parts[2])
	if err != nil {
		return errKeyFormat
	}

	return nil
}

// validateKey takes in key and returns an error if the key is invalid -
// incorrect format, expiration date does not exist, or does not pass the key
// check algorithm
func validateKey(key string) error {
	err := checkKeyFormat(key)
	if err != nil {
		return err
	}

	parts := strings.Split(key, "-")

	// Makes sure that the expiration date exists
	year, _ := strconv.Atoi(parts[0][0:2])
	month, _ := strconv.Atoi(parts[0][2:4])
	day, _ := strconv.Atoi(parts[0][4:6])
	year, err = crypto.Cipher(year, yearCipher, 2)
	if err != nil {
		return errInvalidKey()
	}
	year += 2000
	month, err = crypto.Cipher(month, monthCipher, 2)
	if err != nil {
		return errInvalidKey()
	}
	day, err = crypto.Cipher(day, dayCipher, 2)
	if err != nil {
		return errInvalidKey()
	}
	if !date.DateExists(year, month, day) {
		return errInvalidKey()
	}

	return keyCheckAlgorithm(key)
}

// getExpirationDate returns the expiration date from a valid key This requires
// the key to be in a valid format with a valid date.
func getExpirationDate(key string) time.Time {
	parts := strings.Split(key, "-")

	year, _ := strconv.Atoi(parts[0][0:2])
	month, _ := strconv.Atoi(parts[0][2:4])
	day, _ := strconv.Atoi(parts[0][4:6])
	year, _ = crypto.Cipher(year, yearCipher, 2)
	year += 2000
	month, _ = crypto.Cipher(month, monthCipher, 2)
	day, _ = crypto.Cipher(day, dayCipher, 2)

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
}

// getNumberOfChannels returns the number of channels allowed by key. This
// assumes that the key is in a valid format with a valid date.
func getNumberOfChannels(key string) int64 {
	channelCipher := 64317284
	parts := strings.Split(key, "-")
	numChannels, _ := strconv.Atoi(parts[1])
	numChannels, _ = crypto.Cipher(numChannels, channelCipher, 8)
	return int64(numChannels)
}

// keyCheckAlgorithm takes  key and determines if it is valid. It will return an
// error if the key is not a valid key. This function requires the key to be of
// a valid format with a valid date.
func keyCheckAlgorithm(key string) error {
	code := strings.Split(key, "-")[2]

	var digits [10]int
	for i := 0; i < 10; i++ {
		digits[i], _ = strconv.Atoi(string(code[i]))
	}
	firstFive, _ := strconv.Atoi(code[0:5])
	sum := digits[5] + digits[6] + digits[7] + digits[8]

	if digits[1] != 4 {
		return errInvalidKey()
	} else if firstFive%9 != 0 {
		return errInvalidKey()
	} else if sum%7 != 0 {
		return errInvalidKey()
	} else if digits[9] > 6 || digits[9] < 3 {
		return errInvalidKey()
	}
	return nil
}

func errInvalidKey() error {
	encoded := "aW52YWxpZCBwcm9kdWN0IGxpY2Vuc2Uga2V5"
	errInvalidKeyMsg, _ := base64.StdEncoding.DecodeString(encoded)
	return errors.New(string(errInvalidKeyMsg))
}
