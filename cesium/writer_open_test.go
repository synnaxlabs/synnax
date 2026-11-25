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
	"context"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

const gateTimeout = 30 * time.Second

var domainDataFile = regexp.MustCompile(`^\d+\.domain$`)

// fileGate holds opens of domain data files while it is armed.
type fileGate struct {
	armed   atomic.Bool
	entered chan struct{}
	release chan struct{}
	open    func()
}

func newFileGate() *fileGate {
	g := &fileGate{entered: make(chan struct{}, 1), release: make(chan struct{})}
	g.open = sync.OnceFunc(func() { close(g.release) })
	return g
}

// gates holds a fileGate for each channel directory.
type gates struct {
	byDir map[string]*fileGate
	sync.Mutex
}

func (g *gates) get(dir string) *fileGate {
	g.Lock()
	defer g.Unlock()
	gate, ok := g.byDir[dir]
	if !ok {
		gate = newFileGate()
		g.byDir[dir] = gate
	}
	return gate
}

// gatedFS gives each channel directory its own fileGate.
type gatedFS struct {
	fs.FS
	gate  *fileGate
	gates *gates
}

func newGatedFS(base fs.FS) *gatedFS {
	return &gatedFS{FS: base, gates: &gates{byDir: make(map[string]*fileGate)}}
}

func (g *gatedFS) gateFor(key cesium.ChannelKey) *fileGate {
	return g.gates.get(strconv.Itoa(int(key)))
}

func (g *gatedFS) Sub(name string) (fs.FS, error) {
	sub, err := g.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return &gatedFS{FS: sub, gate: g.gates.get(name), gates: g.gates}, nil
}

func (g *gatedFS) park(name string) error {
	if g.gate == nil || !g.gate.armed.Load() || !domainDataFile.MatchString(name) {
		return nil
	}
	select {
	case g.gate.entered <- struct{}{}:
	default:
	}
	select {
	case <-g.gate.release:
		return nil
	case <-time.After(gateTimeout):
		return errors.New("file gate was not released")
	}
}

func (g *gatedFS) Open(name string, flag int) (fs.File, error) {
	if err := g.park(name); err != nil {
		return nil, err
	}
	return g.FS.Open(name, flag)
}

func (g *gatedFS) Stat(name string) (fs.FileInfo, error) {
	if err := g.park(name); err != nil {
		return nil, err
	}
	return g.FS.Stat(name)
}

var _ = Describe("Writer open", func() {
	Describe("Blocked in the file system", func() {
		const stallTimeout = 5 * time.Second
		var (
			db     *cesium.DB
			gate   *fileGate
			idxKey cesium.ChannelKey
			opened chan error
		)

		BeforeEach(func(ctx SpecContext) {
			gfs := newGatedFS(FileSystems["memFS"]())
			db = mustOpenDBOnFS(ctx, gfs)
			idxKey = GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key:      idxKey,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			})).To(Succeed())
			gate = gfs.gateFor(idxKey)
			gate.armed.Store(true)
			DeferCleanup(gate.open)
			opened = make(chan error, 1)
			go func() {
				w, err := db.OpenWriter(context.Background(), cesium.WriterConfig{
					Channels: []cesium.ChannelKey{idxKey},
					Clock:    idxKey,
					Start:    30 * telem.SecondTS,
				})
				if err == nil {
					err = w.Close()
				}
				opened <- err
			}()
			Eventually(gate.entered).Should(Receive())
			DeferCleanup(func() {
				gate.open()
				Eventually(opened).Should(Receive(Succeed()))
			})
		})

		It("Should let a channel be created", func(ctx SpecContext) {
			created := make(chan error, 1)
			go func() {
				created <- db.CreateChannel(ctx, cesium.Channel{
					Key:      GenerateChannelKey(),
					Name:     "created",
					IsIndex:  true,
					DataType: telem.TimestampT,
				})
			}()
			Eventually(created, stallTimeout).Should(Receive(Succeed()))
		})

		It("Should let another channel be dropped", func(ctx SpecContext) {
			other := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key:      other,
				Name:     "other",
				IsIndex:  true,
				DataType: telem.TimestampT,
			})).To(Succeed())
			dropped := make(chan error, 1)
			go func() { dropped <- db.Drop(ctx, []cesium.ChannelKey{other}) }()
			Eventually(dropped, stallTimeout).Should(Receive(Succeed()))
		})
	})

	Describe("A dependent writer blocked in the file system", func() {
		It("Should refuse to delete the index under it", func(ctx SpecContext) {
			var (
				gfs     = newGatedFS(FileSystems["memFS"]())
				db      = mustOpenDBOnFS(ctx, gfs)
				idxKey  = GenerateChannelKey()
				dataKey = GenerateChannelKey()
			)
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key:      idxKey,
					Name:     "time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      dataKey,
					Name:     "data",
					Index:    idxKey,
					DataType: telem.Int64T,
				},
			)).To(Succeed())
			Expect(writeSeries(
				ctx,
				db,
				idxKey,
				10*telem.SecondTS,
				telem.NewSeriesSecondsTSV(10, 11, 12, 13),
			)).To(Succeed())
			gate := gfs.gateFor(dataKey)
			gate.armed.Store(true)
			DeferCleanup(gate.open)
			opened := make(chan *cesium.Writer, 1)
			go func() {
				defer GinkgoRecover()
				opened <- MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: []cesium.ChannelKey{dataKey},
					Clock:    idxKey,
					Start:    10 * telem.SecondTS,
				}))
			}()
			Eventually(gate.entered).Should(Receive())
			deleted := make(chan error, 1)
			go func() {
				deleted <- db.DeleteTimeRange(
					ctx,
					[]cesium.ChannelKey{idxKey},
					(10 * telem.SecondTS).Range(12*telem.SecondTS),
				)
			}()
			Consistently(deleted, 200*time.Millisecond).ShouldNot(Receive())
			gate.open()
			var w *cesium.Writer
			Eventually(opened).Should(Receive(&w))
			Eventually(deleted).Should(Receive(MatchError(
				ContainSubstring("depending on it"),
			)))
			Expect(w.Close()).To(Succeed())
		})
	})
})
