// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fault_test

import (
	"bytes"
	"fmt"
	"maps"
	"math/rand/v2"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// runKeys returns one table entry for each of 32 run keys.
func runKeys() []TableEntry {
	entries := make([]TableEntry, 32)
	for key := range entries {
		entries[key] = Entry(fmt.Sprintf("run key %d", key), uint64(key))
	}
	return entries
}

// dump returns the bytes of each file under dir in fs by path. A directory has its path
// with a slash at the end, and no bytes.
func dump(fs xfs.FS, dir string, out map[string]string) map[string]string {
	GinkgoHelper()
	for _, info := range MustSucceed(fs.List(dir)) {
		p := path.Join(dir, info.Name())
		if info.IsDir() {
			out[p+"/"] = ""
			dump(fs, p, out)
		} else {
			out[p] = read(fs, p)
		}
	}
	return out
}

// letters returns one random letter, from 1 to 9000 times.
func letters(rng *rand.Rand) []byte {
	return bytes.Repeat([]byte{byte('a' + rng.IntN(26))}, 1+rng.IntN(9000))
}

// churn makes count random changes to files and directories in fs, and calls after
// behind each one. The run key fixes the changes.
func churn(fs xfs.FS, runKey uint64, count int, after func()) {
	GinkgoHelper()
	rng := rand.New(rand.NewPCG(runKey, 0))
	dirs, names := []string{"", "a", "a/b", "c"}, []string{"f", "g", "h"}
	pick := func(from []string) string { return from[rng.IntN(len(from))] }
	file := func() string { return path.Join(pick(dirs), pick(names)) }
	exist := func(name string) bool {
		GinkgoHelper()
		return MustSucceed(fs.Exists(name))
	}
	for range count {
		name := file()
		switch op := rng.IntN(10); {
		case op == 0:
			MustSucceed(fs.Sub(pick(dirs)))
		case op == 1 && exist(name):
			if dir := path.Dir(name); dir != "." && rng.IntN(2) == 0 {
				name = dir
			}
			Expect(fs.Remove(name)).To(Succeed())
		case op == 2 && exist(name):
			if to := file(); exist(path.Dir(to)) {
				Expect(fs.Rename(name, to)).To(Succeed())
			}
		case op == 3:
			if dir := pick(dirs); exist(dir) {
				Expect(fs.SyncDir(dir)).To(Succeed())
			}
		case exist(path.Dir(name)):
			f := MustSucceed(fs.Open(name, os.O_CREATE|os.O_WRONLY))
			switch op {
			case 4:
				Expect(f.Truncate(rng.Int64N(20000))).To(Succeed())
			case 5:
				Expect(f.Sync()).To(Succeed())
			default:
				data := letters(rng)
				Expect(f.WriteAt(data, rng.Int64N(16000))).To(Equal(len(data)))
			}
			Expect(f.Close()).To(Succeed())
		}
		after()
	}
}

var _ = Describe("Properties", func() {
	const blockSize = 4096
	var fs *fault.FS
	BeforeEach(func() { fs = MustSucceed(fault.New(fault.Config{})) })

	DescribeTable("Should give the live state when all of it survives",
		func(runKey uint64) {
			all := fault.CrashConfig{DataSurvivalRate: 1, EntrySurvivalRate: 1}
			churn(fs, runKey, 96, func() {
				image := MustSucceed(fs.Crash(all))
				Expect(dump(image, "", map[string]string{})).
					To(Equal(dump(fs, "", map[string]string{})))
			})
		},
		runKeys(),
	)

	DescribeTable("Should give equal images for equal run keys",
		func(runKey uint64) {
			cfg := fault.CrashConfig{
				RunKey:            runKey,
				DataSurvivalRate:  0.5,
				TornRate:          0.5,
				EntrySurvivalRate: 0.5,
			}
			other := MustSucceed(fault.New(fault.Config{}))
			churn(fs, runKey, 96, func() {})
			churn(other, runKey, 96, func() {})
			image := dump(MustSucceed(fs.Crash(cfg)), "", map[string]string{})
			Expect(dump(MustSucceed(other.Crash(cfg)), "", map[string]string{})).
				To(Equal(image))
		},
		runKeys(),
	)

	DescribeTable("Should keep each file that is durable and live under one name",
		func(runKey uint64) {
			rng := rand.New(rand.NewPCG(runKey, 2))
			names := []string{"a", "b", "c", "d"}
			// contents returns the bytes of each file in the root of image.
			contents := func(image xfs.FS) []string {
				GinkgoHelper()
				return slices.Collect(maps.Values(dump(image, "", map[string]string{})))
			}
			for i := range 64 {
				name := names[rng.IntN(len(names))]
				exists := MustSucceed(fs.Exists(name))
				switch op := rng.IntN(6); {
				case op == 0:
					Expect(fs.SyncDir("")).To(Succeed())
				case op == 1:
					Expect(fs.Remove(name)).To(Succeed())
				case op == 2 && exists:
					Expect(fs.Rename(name, names[rng.IntN(len(names))])).To(Succeed())
				case !exists:
					save(fs, name, strconv.Itoa(i))
				}
				cfg := fault.CrashConfig{RunKey: rng.Uint64(), EntrySurvivalRate: 0.5}
				image := contents(MustSucceed(fs.Crash(cfg)))
				Expect(set.New(image...)).To(HaveLen(len(image)))
				live := contents(fs)
				for _, id := range contents(strict(fs)) {
					if slices.Contains(live, id) {
						Expect(image).To(ContainElement(id))
					}
				}
			}
		},
		runKeys(),
	)

	DescribeTable("Should build each block from its live and its durable bytes",
		func(runKey uint64) {
			rng := rand.New(rand.NewPCG(runKey, 1))
			f := MustOpen(fs.Open("f.bin", os.O_CREATE|os.O_WRONLY))
			Expect(fs.SyncDir("")).To(Succeed())
			// block returns the block at off in data, with zeros past the end of data.
			block := func(data string, off int) string {
				data += strings.Repeat("\x00", blockSize)
				return data[min(off, len(data)-blockSize):][:blockSize]
			}
			for range 48 {
				switch rng.IntN(8) {
				case 0:
					Expect(f.Sync()).To(Succeed())
				case 1:
					Expect(f.Truncate(rng.Int64N(6 * blockSize))).To(Succeed())
				default:
					data := letters(rng)
					Expect(
						f.WriteAt(data, rng.Int64N(5*blockSize)),
					).To(Equal(len(data)))
				}
				live, durable := read(fs, "f.bin"), read(strict(fs), "f.bin")
				cfg := fault.CrashConfig{
					RunKey:           rng.Uint64(),
					DataSurvivalRate: 0.5,
					TornRate:         0.5,
				}
				image := read(MustSucceed(fs.Crash(cfg)), "f.bin")
				Expect(len(image)).To(And(
					BeNumerically(">=", min(len(live), len(durable))),
					BeNumerically("<=", max(len(live), len(durable))),
				))
				for off := 0; off < len(image); off += blockSize {
					size := min(blockSize, len(image)-off)
					got, now := block(image, off)[:size], block(live, off)[:size]
					before := block(durable, off)[:size]
					kept := 0
					for kept < size && got[kept] == now[kept] {
						kept++
					}
					Expect(got[kept:]).To(Equal(before[kept:]), "block at %d", off)
				}
			}
		},
		runKeys(),
	)
})
