// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage_test

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/storage"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Rewrite of version 0 segments", func() {
	DescribeTable(
		"Should let the compaction rewrite a converted old file only when no "+
			"backup is older than the first release that writes segments",
		func(ctx SpecContext, backups []string, backupDisabled, rewritten bool) {
			dir := newDataDir()
			writeOld(xfs.Default, dir, 1, 10, 20)
			Expect(os.MkdirAll(filepath.Join(dir, "backups"), 0o700)).To(Succeed())
			for _, name := range backups {
				writeFile(archiveFile(dir, name), "")
			}
			l, closeLayer := MustSucceed2(openOnLog(ctx, storage.LayerConfig{
				Dirname:              dir,
				Version:              "0.63.0",
				UpdateBackupDisabled: new(backupDisabled),
			}))
			keyDir := filepath.Join(dir, "cesium", "2")
			checks := func() int {
				n := 0
				for _, e := range MustSucceed(os.ReadDir(keyDir)) {
					if strings.HasSuffix(e.Name(), ".chk") {
						n++
					}
				}
				return n
			}
			Eventually(checks).WithTimeout(10 * time.Second).Should(Equal(1))
			// The delete leaves 3 of the 9 samples of the old file of dataCh.
			Expect(l.TS.DeleteTimeRange(
				ctx,
				[]cesium.ChannelKey{dataCh.Key},
				telem.TimeRange{Start: telem.SecondTS, End: 20 * telem.SecondTS},
			)).To(Succeed())
			oldFile := filepath.Join(keyDir, "1.domain")
			exists := func() bool {
				_, err := os.Stat(oldFile)
				return err == nil
			}
			if rewritten {
				Eventually(exists).WithTimeout(10 * time.Second).Should(BeFalse())
			} else {
				Consistently(exists).Should(BeTrue())
			}
			Expect(closeLayer()).To(Succeed())
		},
		Entry("with the backup of the update from 0.62.0", nil, false, false),
		Entry("with no backup", nil, true, true),
		Entry(
			"with a backup of no version",
			[]string{"1700000000-unknown"},
			true,
			false,
		),
		Entry("with a backup of 0.63.0", []string{"1700000000-0.63.0"}, true, true),
	)
})
