// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package timer_test

import (
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/arc/internal/timer"
	. "github.com/synnaxlabs/x/testutil"
)

// cpuTime returns the CPU time the process has used.
func cpuTime() time.Duration {
	var usage syscall.Rusage
	Expect(syscall.Getrusage(syscall.RUSAGE_SELF, &usage)).To(Succeed())
	return time.Duration(usage.Utime.Nano() + usage.Stime.Nano())
}

var _ = Describe("Timer CPU", func() {
	It("Should spin for at most half of each wait", func() {
		s := MustOpen(timer.New(time.Hour))
		const span = 2 * time.Millisecond
		cpu, wall := cpuTime(), timer.Now()
		for range 200 {
			Expect(s.Reset(span)).To(Succeed())
			Eventually(s.C).WithTimeout(time.Second).Should(Receive())
		}
		used := float64(cpuTime()-cpu) / float64(timer.Now()-wall)
		Expect(used).To(BeNumerically("<", 0.75))
	})
})
