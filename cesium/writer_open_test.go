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
	"github.com/synnaxlabs/cesium/internal/resource"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
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

// gatedFS gives each channel directory its own fileGate.
type gatedFS struct {
	fs.FS
	gate  *fileGate
	gates *sync.Map
}

func newGatedFS(base fs.FS) *gatedFS { return &gatedFS{FS: base, gates: &sync.Map{}} }

func (g *gatedFS) gateFor(key cesium.ChannelKey) *fileGate {
	gate, _ := g.gates.LoadOrStore(strconv.Itoa(int(key)), newFileGate())
	return gate.(*fileGate)
}

func (g *gatedFS) Sub(name string) (fs.FS, error) {
	sub, err := g.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	gate, _ := g.gates.LoadOrStore(name, newFileGate())
	return &gatedFS{FS: sub, gate: gate.(*fileGate), gates: g.gates}, nil
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
					Channels:       []cesium.ChannelKey{idxKey},
					Start:          30 * telem.SecondTS,
					Mode:           cesium.WriterModePersistOnly,
					ControlSubject: control.Subject{Key: "blocked"},
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

		It("Should make a time range delete wait with no lock held", func(
			ctx SpecContext,
		) {
			deleted := make(chan error, 1)
			go func() {
				deleted <- db.DeleteTimeRange(
					ctx,
					[]cesium.ChannelKey{idxKey},
					(10 * telem.SecondTS).Range(20*telem.SecondTS),
				)
			}()
			Consistently(deleted, 200*time.Millisecond).ShouldNot(Receive())
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key:      GenerateChannelKey(),
				Name:     "created",
				IsIndex:  true,
				DataType: telem.TimestampT,
			})).To(Succeed())
			gate.open()
			Eventually(deleted, stallTimeout).Should(Receive(Succeed()))
		})

		It("Should report control states", func() {
			states := make(chan cesium.ControlUpdate, 1)
			go func() { states <- db.ControlStates() }()
			Eventually(states, stallTimeout).Should(Receive())
		})

		It("Should fail to delete the channel", func() {
			deleted := make(chan error, 1)
			go func() { deleted <- db.DeleteChannel(idxKey) }()
			Eventually(deleted, stallTimeout).Should(
				Receive(MatchError(resource.ErrOpen)),
			)
		})
	})

})
