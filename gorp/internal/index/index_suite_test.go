// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

func TestIndex(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Index Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// scope is a transaction that runs its hooks when the spec commits or ends it.
type scope struct {
	beforeCommit []func()
	cleanups     []func(committed bool)
}

func (s *scope) BeforeCommit(fn func()) { s.beforeCommit = append(s.beforeCommit, fn) }

func (s *scope) OnCleanup(fn func(committed bool)) {
	s.cleanups = append(s.cleanups, fn)
}

// commit starts a commit of the scope that has not yet ended.
func (s *scope) commit() {
	for _, fn := range s.beforeCommit {
		fn()
	}
	s.beforeCommit = nil
}

func (s *scope) end(committed bool) {
	for _, fn := range s.cleanups {
		fn(committed)
	}
	s.cleanups = nil
}
