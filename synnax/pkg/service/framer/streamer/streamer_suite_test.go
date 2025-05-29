// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

<<<<<<<< HEAD:x/go/bit/bit_suite_test.go
package bit_test
========
package streamer_test
>>>>>>>> 413a05fe0ab771a45f9260dc04bb6d5356fa0034:synnax/pkg/service/framer/streamer/streamer_suite_test.go

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

<<<<<<<< HEAD:x/go/bit/bit_suite_test.go
func TestBit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bit Suite")
========
var ctx = context.Background()

func TestStreamer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Streamer Suite")
>>>>>>>> 413a05fe0ab771a45f9260dc04bb6d5356fa0034:synnax/pkg/service/framer/streamer/streamer_suite_test.go
}
