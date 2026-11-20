// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: prepare after a failed sync", func() {
	for key := range uint64(32) {
		It("Should leave a durable log file of its size once Prepare succeeds after "+
			"a failed sync, run key "+itoa(int(key)), func() {
			disk := MustSucceed(fault.New(fault.Config{RunKey: key}))
			fs := &headFault{FS: disk, failSync: log.Name(4)}
			Expect(checkpoint.Prepare(fs, 4, logSize)).To(MatchError(errSync))
			Expect(checkpoint.Prepare(fs, 4, logSize)).To(Succeed())
			image := MustSucceed(disk.Crash(fault.CrashConfig{
				RunKey:            key,
				DataSurvivalRate:  0.5,
				TornRate:          0.2,
				EntrySurvivalRate: 0.5,
			}))
			Expect(MustSucceed(image.Exists(log.Name(4)))).To(BeTrue())
			Expect(MustSucceed(image.Stat(log.Name(4))).Size()).
				To(Equal(int64(logSize)))
		})

		It("Should leave a durable log file of its size once Prepare succeeds, run "+
			"key "+itoa(int(key)), func() {
			disk := MustSucceed(fault.New(fault.Config{RunKey: key}))
			Expect(checkpoint.Prepare(disk, 4, logSize)).To(Succeed())
			image := MustSucceed(disk.Crash(fault.CrashConfig{
				RunKey:            key,
				DataSurvivalRate:  0.5,
				TornRate:          0.2,
				EntrySurvivalRate: 0.5,
			}))
			Expect(MustSucceed(image.Exists(log.Name(4)))).To(BeTrue())
			Expect(MustSucceed(image.Stat(log.Name(4))).Size()).
				To(Equal(int64(logSize)))
		})
	}
})
