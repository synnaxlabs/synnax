// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package key

import (
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

// Format returns the text form of k: the string itself for a string key, base 10 for
// an integer key, the 8-4-4-4-12 hex form for a 16-byte key, and plain hex for any
// other fixed-size array.
func Format[K primitive](k K) string {
	v := reflect.ValueOf(k)
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Array:
		b := make([]byte, v.Len())
		reflect.Copy(reflect.ValueOf(b), v)
		if len(b) == 16 {
			return formatUUID(b)
		}
		return hex.EncodeToString(b)
	default:
		panic("[gorp] - unsupported key kind " + v.Kind().String())
	}
}

// Parse returns the key whose text form, as Format writes it, is s.
func Parse[K primitive](s string) (K, error) {
	var k K
	v := reflect.ValueOf(&k).Elem()
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, v.Type().Bits())
		if err != nil {
			return k, parseErr(s, err)
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr:
		n, err := strconv.ParseUint(s, 10, v.Type().Bits())
		if err != nil {
			return k, parseErr(s, err)
		}
		v.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, v.Type().Bits())
		if err != nil {
			return k, parseErr(s, err)
		}
		v.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return k, parseErr(s, err)
		}
		v.SetBool(b)
	case reflect.Array:
		b, err := hex.DecodeString(strings.ReplaceAll(s, "-", ""))
		if err != nil {
			return k, parseErr(s, err)
		}
		if len(b) != v.Len() {
			return k, parseErr(s, errors.Newf("expected %d bytes", v.Len()))
		}
		reflect.Copy(v, reflect.ValueOf(b))
	default:
		panic("[gorp] - unsupported key kind " + v.Kind().String())
	}
	return k, nil
}

func parseErr(s string, err error) error {
	return errors.Wrapf(err, "[gorp] - invalid key %q", s)
}

func formatUUID(b []byte) string {
	var buf [36]byte
	hex.Encode(buf[:8], b[:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], b[10:])
	return string(buf[:])
}
