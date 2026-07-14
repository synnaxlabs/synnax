// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"bytes"
	"math"
	"sort"
	"strconv"

	"github.com/synnaxlabs/x/errors"
)

// Canonical encodes v as canonical JSON: object keys sorted lexicographically,
// compact separators, minimal string escaping, and ES6 number formatting. The
// Go, C++, and TypeScript implementations produce identical bytes for the same
// value, making the output suitable for cross-language hashing. Values must be
// JSON-representable (nil, bool, string, integer, float, []any, map[string]any);
// NaN and infinities are rejected.
func Canonical(v any) ([]byte, error) {
	var b bytes.Buffer
	if err := writeCanonical(&b, v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func writeCanonical(b *bytes.Buffer, v any) error {
	switch val := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if val {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case string:
		writeCanonicalString(b, val)
	case int:
		b.WriteString(strconv.FormatInt(int64(val), 10))
	case int8:
		b.WriteString(strconv.FormatInt(int64(val), 10))
	case int16:
		b.WriteString(strconv.FormatInt(int64(val), 10))
	case int32:
		b.WriteString(strconv.FormatInt(int64(val), 10))
	case int64:
		b.WriteString(strconv.FormatInt(val, 10))
	case uint:
		b.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint8:
		b.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint16:
		b.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint32:
		b.WriteString(strconv.FormatUint(uint64(val), 10))
	case uint64:
		b.WriteString(strconv.FormatUint(val, 10))
	case float32:
		return writeES6Number(b, float64(val))
	case float64:
		return writeES6Number(b, val)
	case []any:
		b.WriteByte('[')
		for i, elem := range val {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := writeCanonical(b, elem); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case map[string]any:
		return writeCanonicalObject(b, val)
	case map[any]any:
		m := make(map[string]any, len(val))
		for k, elem := range val {
			ks, ok := k.(string)
			if !ok {
				return errors.Newf("canonical JSON: non-string object key %v", k)
			}
			m[ks] = elem
		}
		return writeCanonicalObject(b, m)
	default:
		return errors.Newf("canonical JSON: unsupported type %T", v)
	}
	return nil
}

func writeCanonicalObject(b *bytes.Buffer, m map[string]any) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		writeCanonicalString(b, k)
		b.WriteByte(':')
		if err := writeCanonical(b, m[k]); err != nil {
			return err
		}
	}
	b.WriteByte('}')
	return nil
}

const hexDigits = "0123456789abcdef"

func writeCanonicalString(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for i := range len(s) {
		switch c := s[i]; c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if c < 0x20 {
				b.WriteString(`\u00`)
				b.WriteByte(hexDigits[c>>4])
				b.WriteByte(hexDigits[c&0xf])
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
}

// writeES6Number formats f per the ECMAScript Number::toString algorithm, the
// format JSON.stringify emits. Integral doubles print without a decimal point,
// so 50.0 and 50 canonicalize identically.
func writeES6Number(b *bytes.Buffer, f float64) error {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return errors.New("canonical JSON: NaN and Inf are not representable")
	}
	if f == 0 {
		b.WriteByte('0')
		return nil
	}
	if f < 0 {
		b.WriteByte('-')
		f = -f
	}
	// Shortest round-trip digits in the form d[.ddd]e±XX.
	sci := strconv.FormatFloat(f, 'e', -1, 64)
	eIdx := bytes.IndexByte([]byte(sci), 'e')
	mant := sci[:eIdx]
	exp10, err := strconv.Atoi(sci[eIdx+1:])
	if err != nil {
		return errors.Wrap(err, "canonical JSON: malformed float exponent")
	}
	digits := mant[:1]
	if len(mant) > 2 {
		digits += mant[2:]
	}
	k, n := len(digits), exp10+1
	switch {
	case k <= n && n <= 21:
		b.WriteString(digits)
		for range n - k {
			b.WriteByte('0')
		}
	case 0 < n && n <= 21:
		b.WriteString(digits[:n])
		b.WriteByte('.')
		b.WriteString(digits[n:])
	case -6 < n && n <= 0:
		b.WriteString("0.")
		for range -n {
			b.WriteByte('0')
		}
		b.WriteString(digits)
	default:
		b.WriteString(digits[:1])
		if k > 1 {
			b.WriteByte('.')
			b.WriteString(digits[1:])
		}
		b.WriteByte('e')
		if n-1 >= 0 {
			b.WriteByte('+')
		}
		b.WriteString(strconv.Itoa(n - 1))
	}
	return nil
}
