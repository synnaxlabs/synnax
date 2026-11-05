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
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/resource"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
)

var domainDataFile = regexp.MustCompile(`^\d+\.domain$`)

// gatedFS holds each open of a domain data file until release closes.
type gatedFS struct {
	fs.FS
	entered chan struct{}
	release chan struct{}
}

func (g *gatedFS) Sub(name string) (fs.FS, error) {
	sub, err := g.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return &gatedFS{FS: sub, entered: g.entered, release: g.release}, nil
}

func (g *gatedFS) Open(name string, flag int) (fs.File, error) {
	if domainDataFile.MatchString(name) {
		select {
		case g.entered <- struct{}{}:
		default:
		}
		<-g.release
	}
	return g.FS.Open(name, flag)
}

var _ = Describe("Writer open", func() {
	Describe("Blocked in the file system", func() {
		const stallTimeout = 5 * time.Second
		var (
			db      *cesium.DB
			gfs     *gatedFS
			idxKey  cesium.ChannelKey
			release func()
			opened  chan error
		)

		BeforeEach(func(ctx SpecContext) {
			gfs = &gatedFS{
				FS:      FileSystems["memFS"](),
				entered: make(chan struct{}, 1),
				release: make(chan struct{}),
			}
			release = sync.OnceFunc(func() { close(gfs.release) })
			DeferCleanup(release)
			db = mustOpenDBOnFS(ctx, gfs)
			idxKey = GenerateChannelKey()
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key:      idxKey,
				Name:     "time",
				IsIndex:  true,
				DataType: telem.TimestampT,
			})).To(Succeed())
			opened = make(chan error, 1)
			go func() {
				w, err := db.OpenWriter(context.Background(), cesium.WriterConfig{
					Channels:       []cesium.ChannelKey{idxKey},
					Start:          telem.SecondTS,
					Mode:           cesium.WriterModePersistOnly,
					ControlSubject: control.Subject{Key: "blocked"},
				})
				if err == nil {
					err = w.Close()
				}
				opened <- err
			}()
			Eventually(gfs.entered).Should(Receive())
			DeferCleanup(func() {
				release()
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

		It("Should fail to delete a time range it covers", func(ctx SpecContext) {
			deleted := make(chan error, 1)
			go func() {
				deleted <- db.DeleteTimeRange(
					ctx,
					[]cesium.ChannelKey{idxKey},
					telem.TimeRangeMax,
				)
			}()
			Eventually(deleted, stallTimeout).Should(
				Receive(MatchError(control.ErrUnauthorized)),
			)
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
