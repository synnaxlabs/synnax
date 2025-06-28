// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

<<<<<<<< HEAD:x/go/zyn/literal.go
package zyn

func Literal[T comparable](value T) EnumZ { return Enum(value) }
========
package zyn_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestZyn(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Zyn Suite")
}
>>>>>>>> d0e4900de158a437b73160b89b1f0eaf95b4254e:x/go/zyn/zyn_suite_test.go
