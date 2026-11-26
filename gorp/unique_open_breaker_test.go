// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// refBreakerItem is the entry of a table that references a folder and fails to open.
type refBreakerItem struct {
	Key    uint32
	Folder string
	Name   string
}

func (i refBreakerItem) GorpKey() uint32 { return i.Key }

var _ = Describe("UniqueIndex failed open", func() {
	It(
		"Should not hang a delete of a table that a failed open references",
		func(ctx SpecContext) {
			gated := &gatedKV{
				DB:      memkv.New(),
				match:   []byte("refBreakerItem"),
				held:    make(chan struct{}),
				release: make(chan struct{}),
			}
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(gated.release) }) }
			DeferCleanup(release)
			db := DeferClose(gorp.Wrap(gated, gorp.WithCodec(msgpack.Codec)))
			StartRun(db.Run)
			folders := MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[string, uniqueFolder]{
					DB:      db,
					Name:    "folder",
					Indexes: []gorp.Index[string, uniqueFolder]{newFolderIndex()},
				}),
			)
			Expect(folders.NewCreate().
				Entry(&uniqueFolder{Key: "a", Name: "a"}).
				Exec(ctx, db)).To(Succeed())
			sibling := func(i *refBreakerItem) (uniqueSibling, bool) {
				return uniqueSibling{Parent: i.Folder, Name: i.Name}, true
			}
			opened := make(chan error, 1)
			go func() {
				// Two indexes of one table in one space: the join fails after the
				// table registers its reference to folder.
				_, err := gorp.OpenTable(ctx, gorp.TableConfig[uint32, refBreakerItem]{
					DB:   db,
					Name: "failing",
					Indexes: []gorp.Index[uint32, refBreakerItem]{
						gorp.NewUniqueIndex("a", uniqueSpaceName, sibling),
						gorp.NewUniqueIndex("b", uniqueSpaceName, sibling),
					},
					References: []gorp.Reference[uint32, refBreakerItem]{
						gorp.NewKeyReference(
							gorp.KeyReferenceConfig[uint32, refBreakerItem, string]{
								Name:     "folder",
								Table:    "folder",
								OnDelete: gorp.Detach,
								Kind:     gorp.Place,
								Targets: func(i *refBreakerItem) []string {
									if i.Folder == "" {
										return nil
									}
									return []string{i.Folder}
								},
								Detach: func(i *refBreakerItem, _ string) {
									i.Folder = ""
								},
							},
						),
					},
				})
				opened <- err
			}()
			// The failed open waits for its populate, which the gate holds.
			Eventually(gated.held).Should(BeClosed())
			timeout, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			deleted := make(chan error, 1)
			go func() {
				deleted <- folders.NewDelete().
					Where(gorp.MatchKeys[string, uniqueFolder]("a")).
					Exec(timeout, db)
			}()
			// The delete walks the references to folder while the open still fails.
			time.Sleep(500 * time.Millisecond)
			release()
			Eventually(opened).Should(Receive(MatchError(ContainSubstring(
				"table failing already has an index",
			))))
			written := make(chan error, 1)
			go func() {
				written <- folders.NewCreate().
					Entry(&uniqueFolder{Key: "b", Name: "b"}).
					Exec(ctx, db)
			}()
			// The failed open has returned, so neither the delete nor a later write
			// waits on it.
			Eventually(deleted).WithTimeout(time.Second).Should(Receive(BeNil()))
			Eventually(written).WithTimeout(time.Second).Should(Receive(BeNil()))
		},
	)
})
