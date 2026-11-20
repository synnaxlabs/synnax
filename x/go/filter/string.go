// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package filter

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// CompileString returns a function that reports whether a value matches every operator
// of s. With CaseInsensitive set, every operator ignores case by Unicode simple case
// folding, the rule of the (?i) regex flag. It returns a validate.PathError at "regex"
// if s.Regex does not compile.
func CompileString(s String) (func(string) bool, error) {
	norm := func(v string) string { return v }
	if s.CaseInsensitive {
		norm = fold
	}
	var checks []func(string) bool
	if s.Eq != nil {
		eq := norm(*s.Eq)
		checks = append(checks, func(v string) bool { return v == eq })
	}
	if s.In != nil {
		in := make([]string, len(s.In))
		for i, e := range s.In {
			in[i] = norm(e)
		}
		checks = append(checks, func(v string) bool { return slices.Contains(in, v) })
	}
	if s.Contains != nil {
		sub := norm(*s.Contains)
		checks = append(checks, func(v string) bool { return strings.Contains(v, sub) })
	}
	if s.StartsWith != nil {
		prefix := norm(*s.StartsWith)
		checks = append(checks, func(v string) bool {
			return strings.HasPrefix(v, prefix)
		})
	}
	var re *regexp.Regexp
	if s.Regex != nil {
		pattern := *s.Regex
		if s.CaseInsensitive {
			pattern = "(?i)" + pattern
		}
		var err error
		if re, err = regexp.Compile(pattern); err != nil {
			return nil, validate.PathedError(
				errors.Wrap(validate.ErrValidation, err.Error()),
				"regex",
			)
		}
	}
	return func(v string) bool {
		if re != nil && !re.MatchString(v) {
			return false
		}
		v = norm(v)
		for _, check := range checks {
			if !check(v) {
				return false
			}
		}
		return true
	}, nil
}

// fold maps each rune of s to one member of its simple case folding orbit, so two
// strings fold equal exactly when (?i) matches them. ASCII folds to lower case, which
// keeps a lower-case ASCII string from allocating.
func fold(s string) string {
	return strings.Map(func(r rune) rune {
		if r < utf8.RuneSelf {
			return unicode.ToLower(r)
		}
		folded := r
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			if f < utf8.RuneSelf {
				return unicode.ToLower(f)
			}
			folded = min(folded, f)
		}
		return folded
	}, s)
}
