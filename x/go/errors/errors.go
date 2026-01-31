// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package errors

import "github.com/cockroachdb/errors"

const (
	// TypeEmpty represents an error that hasn't been properly parsed or detected.
	// TypeEmpty errors typically represent a programming error.
	TypeEmpty = ""
	// TypeNil represents a nil error i.e. one that has not occurred.
	TypeNil = "nil"
	// TypeUnknown represents an error that was not registered with the errors package.
	TypeUnknown = "unknown"
	// TypeRoach represents an error type that was encoded using cockroachdb's errors
	// package. This is the default error type for errors that are not registered with
	// the errors package, and is used mostly for Go-to-Go communication.
	TypeRoach = "roach"
)

// Is determines whether one of the causes of the given error or any of its causes is
// equivalent to some reference error.
//
// As in the Go standard library, an error is considered to match a reference error if
// it is equal to that target or if it implements a method Is(error) bool such that
// Is(reference) returns true
func Is(err, ref error) bool { return errors.Is(err, ref) }

// IsAny determines whether any causes the given error or any of its causes is
// equivalent to one of the reference errors.
//
// Note that IsAny returns true if err matches errors.Is for ANY of the reference errors
// i.e., this is an OR operation, not an AND operation.
//
// As in the Go standard library, an error is considered to match a reference error if
// it is equal to that target or if it implements a method Is(error) bool such that
// Is(reference) returns true
func IsAny(err error, refs ...error) bool {
	for _, e := range refs {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

// Wrap wraps an error with a message prefix. A stack trace is retained
func Wrap(err error, msg string) error { return errors.Wrap(err, msg) }

// Wrapf wraps an error with a formatted message prefix. A stack trace is also retained.
// If the format is empty, no prefix is added, but the extra arguments are still
// processed for reportable strings
func Wrapf(err error, format string, args ...any) error {
	return errors.Wrapf(err, format, args...)
}

// Join returns an error that wraps the given errors. Any nil error values are
// discarded. Join returns nil if errs contains no non-nil values. The error formats as
// the concatenation of the strings obtained by calling the Error method of each element
// of errs, with a newline between each string. A stack trace is also retained.
func Join(errs ...error) error { return errors.Join(errs...) }

// Combine returns:
//  1. err if otherErr is nil.
//  2. nil if both err and otherErr are nil.
//  3. otherErr if err is nil.
//  4. if err and otherErr are non-nil, attaches otherErr as a secondary error. See
//     WithSecondaryError for details on how the secondary error is presented.
func Combine(err error, otherErr error) error {
	return errors.CombineErrors(err, otherErr)
}

// New creates an error with a simple error message. A stack trace is retained.
func New(msg string) error { return WithStackDepth(errors.New(msg), 1) }

// Newf creates an error with a formatted error message. A stack trace is retained.
func Newf(format string, args ...any) error { return errors.Newf(format, args...) }

// As finds the first error in err's chain that matches the type to which target points,
// and if so, sets the target to its value and returns true. An error matches a type if
// it is assignable to the target type, or if it has a method As(any) bool such that
// As(target) returns true. As will panic if target is not a non-nil pointer to a type
// which implements error or is of interface type.
//
// The As method should set the target to its value and return true if err matches the
// type to which target points.
func As(err error, target any) bool { return errors.As(err, target) }

// Skip returns nil if the error satisfied errors.Is for any of the reference errors.
// Otherwise, it returns the error itself.
func Skip(err error, refs ...error) error {
	if errors.IsAny(err, refs...) {
		return nil
	}
	return err
}
