// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

<<<<<<<< HEAD:synnax/pkg/service/framer/iterator/iterator_suite_test.go
package iterator_test
========
package bounds_test
>>>>>>>> 198c29031eeec58c5985ca14967ae9e3884f92e2:x/go/bounds/bounds_suite_test.go

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

<<<<<<<< HEAD:synnax/pkg/service/framer/iterator/iterator_suite_test.go
var ctx = context.Background()

func TestIterator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Iterator Suite")
========
func TestBounds(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bounds Suite")
>>>>>>>> 198c29031eeec58c5985ca14967ae9e3884f92e2:x/go/bounds/bounds_suite_test.go
}
