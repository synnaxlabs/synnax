// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFalamos(t *testing.T) {
	RegisterFailHandler(Fail)
	// This suite creates multiple traced instrumentations in a single spec, each of
	// which reconfigures the process-global OpenTelemetry SDK via uptrace. Only the
	// last-configured provider can be shut down on Close, so the earlier ones' SDK
	// daemons (periodic metric reader, log batch processor) are orphaned and cannot be
	// drained — per-spec goroutine-leak checking is therefore not applicable here.
	//nolint:leaklint
	RunSpecs(t, "Alamos Suite")
}
