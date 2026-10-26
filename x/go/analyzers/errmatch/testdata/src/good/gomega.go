// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package good

type Assertion struct{}

func (Assertion) To(any, ...any) bool        { return true }
func (Assertion) ToNot(any, ...any) bool     { return true }
func (Assertion) ShouldNot(any, ...any) bool { return true }
func (a Assertion) Error() Assertion         { return a }

func Expect(any, ...any) Assertion     { return Assertion{} }
func Eventually(any, ...any) Assertion { return Assertion{} }

func HaveOccurred() any           { return nil }
func Succeed() any                { return nil }
func BeNil() any                  { return nil }
func Not(any) any                 { return nil }
func MatchError(any) any          { return nil }
func ContainSubstring(string) any { return nil }

var errSentinel error

func fails() error       { return nil }
func pair() (int, error) { return 0, nil }
func ptr() *int          { return nil }
