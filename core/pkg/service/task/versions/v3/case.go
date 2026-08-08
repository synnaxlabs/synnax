// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

// camelToSnake is a Go port of the TS wire codec's camelToSnake string conversion, so
// legacy Console-written keys convert exactly as they would have on the wire. A capital
// breaks a word when it follows a lowercase or digit, or when it continues an uppercase
// run entered from one: fooXY (from foo_x_y) splits fully while a leading run like
// NS=1;ID=5 stays put.
func camelToSnake(str string) string {
	isUpper := func(c byte) bool { return c >= 'A' && c <= 'Z' }
	isLowerOrDigit := func(c byte) bool {
		return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
	}
	n := len(str)
	i := 1
	for ; i < n; i++ {
		if isUpper(str[i]) && isLowerOrDigit(str[i-1]) {
			break
		}
	}
	if i >= n {
		return str
	}
	res := make([]byte, 0, n+4)
	res = append(res, str[:i]...)
	enteredFromLower := true
	for ; i < n; i++ {
		c := str[i]
		if !isUpper(c) {
			enteredFromLower = false
			res = append(res, c)
			continue
		}
		prev := str[i-1]
		if isLowerOrDigit(prev) {
			enteredFromLower = true
		} else if !isUpper(prev) {
			enteredFromLower = false
		}
		if enteredFromLower {
			res = append(res, '_', c+('a'-'A'))
		} else {
			res = append(res, c)
		}
	}
	return string(res)
}

// snakeKeys returns a copy of m with every map key recursively converted through
// camelToSnake, including keys of maps nested inside lists. When both spellings of a
// key are present, the snake_case one wins. Values are never converted, even former
// record keys that listification moved into value position. The input and its nested
// structures are left unmodified.
func snakeKeys(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if camelToSnake(k) == k {
			out[k] = snakeKeysValue(v)
		}
	}
	for k, v := range m {
		nk := camelToSnake(k)
		if nk == k {
			continue
		}
		if _, taken := out[nk]; !taken {
			out[nk] = snakeKeysValue(v)
		}
	}
	return out
}

// snakeKeysValue recurses snakeKeys through nested maps and list elements.
func snakeKeysValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return snakeKeys(t)
	case []any:
		out := make([]any, len(t))
		for i, el := range t {
			out[i] = snakeKeysValue(el)
		}
		return out
	}
	return v
}
