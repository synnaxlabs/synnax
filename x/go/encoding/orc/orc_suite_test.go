// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package orc_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// magic is the orc magic header. intentional kept separate from the
// production package definition to ensure format does not unexpectedly get changed.
var magic = [3]byte{0x4F, 0x52, 0x43}

func TestOrc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Orc Suite")
}
