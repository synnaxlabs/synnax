// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

import (
	"github.com/cockroachdb/errors"
)

// IsAny returns true if err is any of the given errors.
func IsAny(err error, errs ...error) bool {
	for _, e := range errs {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

func Is(err, target error) bool { return errors.Is(err, target) }

func Wrap(err error, msg string) error { return errors.Wrap(err, msg) }

func Wrapf(err error, format string, args ...interface{}) error {
	return errors.Wrapf(err, format, args...)
}

// Combine returns err, or, if err is nil, otherErr. if err is non-nil, otherErr
// is attached as secondary error. See the documentation of `WithSecondaryError()` for
// details
func Combine(err error, otherErr error) error {
	return errors.CombineErrors(err, otherErr)
}

func New(msg string) error { return errors.New(msg) }

func Newf(format string, args ...interface{}) error { return errors.Newf(format, args...) }

func As(err error, target interface{}) bool { return errors.As(err, target) }

func Skip(err error, targets ...error) error {
	if errors.IsAny(err, targets...) {
		return nil
	}
	return err
}
