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

func (Assertion) To(any, ...any) bool { return true }

func Expect(any, ...any) Assertion { return Assertion{} }
func Succeed() any                 { return nil }

func It(string, func())        {}
func BeforeEach(func())        {}
func AfterEach(func())         {}
func JustAfterEach(func())     {}
func AfterAll(func())          {}
func AfterSuite(func())        {}
func DeferCleanup(any, ...any) {}

type closer struct{}

func (closer) Close() error { return nil }

// inlet closes a channel, so it is not an io.Closer.
type inlet struct{}

func (inlet) Close() {}

func open() (closer, error) { return closer{}, nil }
