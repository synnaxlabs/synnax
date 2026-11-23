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
	"context"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var errRemove = errors.New("remove failed")

// running runs c until stop, which closes c and returns the error of Close: the error
// of the last head, or of the first clean.
func running(c *checkpoint.Checkpointer) (stop func() error) {
	ran := make(chan error, 1)
	go func() { ran <- c.Run(context.Background()) }()
	return func() error {
		GinkgoHelper()
		err := c.Close()
		Eventually(ran).Should(Receive(BeNil()))
		return err
	}
}

// headFailFS fails the second sync of each head file that fails names, which follows
// the write of its header sector, and each remove of it. refused counts the removes
// that it failed.
type headFailFS struct {
	xfs.FS
	fails   func(name string) bool
	refused *atomic.Int32
}

func (f headFailFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil || !f.fails(name) {
		return file, err
	}
	return &headFailFile{File: file}, nil
}

func (f headFailFS) Remove(name string) error {
	if f.fails(name) {
		if ok, err := f.Exists(name); err != nil || ok {
			f.refused.Add(1)
			return errors.Join(err, errRemove)
		}
	}
	return f.FS.Remove(name)
}

type headFailFile struct {
	xfs.File
	syncs int
}

func (f *headFailFile) Sync() error {
	if f.syncs++; f.syncs == 2 {
		return errSync
	}
	return f.File.Sync()
}

var _ = Describe("Part undo check", func() {
	It("Should keep the parts of a failed head that the switch could not remove",
		func() {
			fs := headFailFS{
				FS:      xfs.NewMem(),
				fails:   func(name string) bool { return strings.HasSuffix(name, ".head") },
				refused: &atomic.Int32{},
			}
			c := MustSucceed(checkpoint.New(checkpoint.Config{
				FS:       fs,
				DB:       db,
				LogSize:  logSize,
				Prepared: func(uint64, error) {},
				Executor: goExec,
				PartSize: 1,
			}))
			stop := running(c)
			c.Head(2, withSealed(sealed(1, 10)))
			Expect(stop()).To(MatchError(errRemove))
			h, _ := MustSucceed2(checkpoint.Read(fs, 2))
			Expect(h.Parts).To(HaveLen(1))
			MustSucceed(checkpoint.LoadParts(fs, db, h))
		})

	It("Should write the next switch after a failed head that the switch could not "+
		"remove", func() {
		fs := headFailFS{
			FS:      xfs.NewMem(),
			fails:   func(name string) bool { return name == checkpoint.Name(2) },
			refused: &atomic.Int32{},
		}
		c := MustSucceed(checkpoint.New(checkpoint.Config{
			FS:       fs,
			DB:       db,
			LogSize:  logSize,
			Prepared: func(uint64, error) {},
			Executor: goExec,
			PartSize: 1,
		}))
		stop := running(c)
		c.Head(2, withSealed(sealed(1, 10)))
		// Head 2 fails once its remove does.
		Eventually(fs.refused.Load).Should(BeNumerically(">", 0))
		c.Head(3, withSealed(sealed(1, 10), sealed(2, 20)))
		Eventually(func() error { return readErr(fs, 3) }).
			WithTimeout(5 * time.Second).
			Should(Succeed())
		h, _ := MustSucceed2(checkpoint.Read(fs, 3))
		MustSucceed(checkpoint.LoadParts(fs, db, h))
		// The clean of switch 3 cannot remove head 2. The next switch tries again.
		Expect(stop()).To(MatchError(errRemove))
	})
})
