// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

// checkArmFS arms the failure of the next log sync when the check pass creates a check
// file.
type checkArmFS struct {
	xfs.FS
	armed *atomic.Bool
}

func (f checkArmFS) Sub(name string) (xfs.FS, error) {
	sub, err := f.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return checkArmFS{FS: sub, armed: f.armed}, nil
}

func (f checkArmFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err == nil && strings.HasSuffix(name, ".chk") && flag&os.O_CREATE != 0 {
		f.armed.Store(true)
	}
	return file, err
}

// oldFilesOf returns the number of old data files with samples of each of keys in fs.
// The conversion makes one version 0 segment of each.
func oldFilesOf(fs xfs.FS, keys ...cesium.ChannelKey) map[cesium.ChannelKey]int {
	GinkgoHelper()
	out := make(map[cesium.ChannelKey]int, len(keys))
	for _, key := range keys {
		out[key] = 0
		for _, info := range MustSucceed(fs.List(strconv.FormatUint(uint64(key), 10))) {
			name := info.Name()
			if strings.HasSuffix(name, ".domain") && name != "index.domain" &&
				name != "counter.domain" && info.Size() > 0 {
				out[key]++
			}
		}
	}
	return out
}

var _ = Describe("Check: a Check whose log sync fails", func() {
	It("Should keep the check file, then check and read every sample after a restart",
		func(ctx SpecContext) {
			keys := []cesium.ChannelKey{oldIdx, oldI32, oldStr}
			disk := MustSucceed(fault.New(fault.Config{}))
			old := writeOld(ctx, disk)
			armed, failed := &atomic.Bool{}, &atomic.Bool{}
			fs := logSyncFS{
				FS:     checkArmFS{FS: disk, armed: armed},
				armed:  armed,
				failed: failed,
			}
			ins := alamos.New("cesium")
			_, closeDB := openConverted(ctx, fs, ins)
			Eventually(failed.Load).WithTimeout(time.Minute).Should(BeTrue())
			Expect(closeDB()).To(MatchError(ContainSubstring(
				"no entry is durable until a restart: disk failed",
			)))
			// The log may hold the Check, so the check file stays for the next open.
			kept := 0
			for _, n := range MustSucceed(checkFiles(disk, keys...)) {
				kept += n
			}
			Expect(kept).To(BeNumerically(">", 0))
			db, closeDB := openConverted(ctx, disk, ins)
			Eventually(func() (map[cesium.ChannelKey]int, error) {
				return checkFiles(disk, keys...)
			}).WithTimeout(time.Minute).Should(Equal(oldFilesOf(disk, keys...)))
			Expect(dataOf(ctx, db, keys)).To(Equal(old.bytes(nil)))
			Expect(closeDB()).To(Succeed())
		},
	)
})
