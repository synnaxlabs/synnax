// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
)

// mapScanner is a driver.Scanner whose check results a spec sets per device.
type mapScanner struct {
	mu sync.Mutex
	// unreachable holds the check error of each device that is not reachable.
	unreachable map[device.Key]error
	checks      map[device.Key]int
	// release ends one wait command.
	release chan struct{}
	// waiting and peak count the wait commands that run now and at most.
	waiting, peak atomic.Int32
}

func newMapScanner() *mapScanner {
	return &mapScanner{
		unreachable: make(map[device.Key]error),
		checks:      make(map[device.Key]int),
		release:     make(chan struct{}),
	}
}

func (s *mapScanner) setUnreachable(key device.Key, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		delete(s.unreachable, key)
		return
	}
	s.unreachable[key] = err
}

func (s *mapScanner) checkCount(key device.Key) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.checks[key]
}

func (s *mapScanner) Check(_ context.Context, dev device.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks[dev.Key]++
	return s.unreachable[dev.Key]
}

func (s *mapScanner) Exec(
	ctx context.Context,
	cmd task.Command,
) (msgpack.EncodedJSON, error) {
	switch cmd.Type {
	case "test_connection":
		return msgpack.EncodedJSON{"latency_ms": 4.0}, nil
	case "browse":
		return nil, errors.New("broker refused the subscription")
	case "wait":
		n := s.waiting.Add(1)
		defer s.waiting.Add(-1)
		if n > s.peak.Load() {
			s.peak.Store(n)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-s.release:
			return msgpack.EncodedJSON{"waited": true}, nil
		}
	}
	return nil, driver.ErrUnsupportedCommand
}

// refusingDB is a kv.DB whose transactions fail to commit when they set a key that
// holds marker.
type refusingDB struct {
	kv.DB
	marker []byte
}

func (d refusingDB) OpenTx() kv.Tx {
	return &refusingTx{Tx: d.DB.OpenTx(), marker: d.marker}
}

type refusingTx struct {
	kv.Tx
	marker  []byte
	refused bool
}

func (t *refusingTx) Set(ctx context.Context, key, value []byte, opts ...any) error {
	t.refused = t.refused || bytes.Contains(key, t.marker)
	return t.Tx.Set(ctx, key, value, opts...)
}

func (t *refusingTx) Commit(ctx context.Context, opts ...any) error {
	if t.refused {
		return errCommit
	}
	return t.Tx.Commit(ctx, opts...)
}

var _ = Describe("ScanTask", func() {
	var (
		t          task.Task
		scanner    *mapScanner
		rackKey    rack.Key
		deviceMake string
	)

	BeforeEach(func(ctx SpecContext) {
		r := rack.Rack{Name: "scan-task-rack"}
		Expect(rackService.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		rackKey = r.Key
		deviceMake = "make-" + uuid.New().String()
		t = task.Task{
			Key:  uuid.New(),
			Name: "scan-task-test",
			Type: "mock",
			Rack: rackKey,
		}
		scanner = newMapScanner()
	})

	openOn := func(statusDB *gorp.DB, interval time.Duration) *driver.ScanTask {
		GinkgoHelper()
		st := MustSucceed(driver.NewScanTask(driver.ScanTaskConfig{
			Scanner: scanner,
			DB:      statusDB, Status: statusSvc,
			Device:             deviceSvc,
			Make:               deviceMake,
			ReachableMessage:   "Broker connected",
			UnreachableMessage: "Failed to reach broker",
			Task:               t,
			Interval:           interval,
		}))
		DeferCleanup(func() { Expect(st.Stop(false)).To(Succeed()) })
		return st
	}
	open := func(interval time.Duration) *driver.ScanTask {
		GinkgoHelper()
		return openOn(db, interval)
	}

	// writeDevices runs write in one transaction, as the API does, so a device and its
	// status change together.
	writeDevices := func(ctx context.Context, write func(device.Writer) error) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return write(deviceSvc.NewWriter(tx))
		})).To(Succeed())
	}

	createDevice := func(ctx context.Context, deviceMake string) device.Device {
		GinkgoHelper()
		dev := device.Device{
			Key:      uuid.New().String(),
			Rack:     rackKey,
			Location: "localhost:1883",
			Name:     "Broker",
			Make:     deviceMake,
			Model:    "broker",
		}
		writeDevices(ctx, func(w device.Writer) error { return w.Create(ctx, &dev) })
		return dev
	}

	deviceStatus := func(ctx context.Context, dev device.Device) device.Status {
		GinkgoHelper()
		return retrieveStatus[device.StatusDetails](ctx, dev.OntologyID().String())
	}

	taskStatus := func(ctx context.Context) task.Status {
		GinkgoHelper()
		return retrieveStatus[task.StatusDetails](ctx, t.OntologyID().String())
	}

	Describe("NewScanTask", func() {
		It("Should reject a configuration with no make", func() {
			Expect(driver.NewScanTask(driver.ScanTaskConfig{
				Scanner: scanner,
				DB:      db, Status: statusSvc,
				Device:             deviceSvc,
				ReachableMessage:   "Broker connected",
				UnreachableMessage: "Failed to reach broker",
				Task:               t,
			})).Error().To(MatchError(ContainSubstring("make")))
		})
	})

	Describe("Device checks", func() {
		It("Should report each device of its make as reachable or not",
			func(ctx SpecContext) {
				up := createDevice(ctx, deviceMake)
				down := createDevice(ctx, deviceMake)
				other := createDevice(ctx, "other-make")
				scanner.setUnreachable(down.Key, errors.New("dial timeout"))
				st := open(time.Hour)
				Expect(st.Start(ctx, driver.NoCommand)).To(Succeed())
				Eventually(func(g Gomega) {
					stat := deviceStatus(ctx, up)
					g.Expect(stat.Variant).To(Equal(status.VariantSuccess))
					g.Expect(stat.Message).To(Equal("Broker connected"))
					g.Expect(stat.Details.Device).To(Equal(up.Key))
					g.Expect(stat.Details.Rack).To(Equal(rackKey))
				}).Should(Succeed())
				stat := deviceStatus(ctx, down)
				Expect(stat.Variant).To(Equal(status.VariantWarning))
				Expect(stat.Message).To(Equal("Failed to reach broker"))
				Expect(stat.Description).To(Equal("dial timeout"))
				Expect(scanner.checkCount(other.Key)).To(BeZero())
			},
		)

		It("Should check a device when it is created", func(ctx SpecContext) {
			st := open(time.Hour)
			Expect(st.Start(ctx, driver.NoCommand)).To(Succeed())
			dev := createDevice(ctx, deviceMake)
			Eventually(func(g Gomega) {
				g.Expect(deviceStatus(ctx, dev).Message).To(Equal("Broker connected"))
			}).Should(Succeed())
		})

		It("Should report a change of health on the next interval",
			func(ctx SpecContext) {
				dev := createDevice(ctx, deviceMake)
				st := open(5 * time.Millisecond)
				Expect(st.Start(ctx, driver.NoCommand)).To(Succeed())
				Eventually(func(g Gomega) {
					g.Expect(deviceStatus(ctx, dev).Variant).
						To(Equal(status.VariantSuccess))
				}).Should(Succeed())
				scanner.setUnreachable(dev.Key, errors.New("connection reset"))
				Eventually(func(g Gomega) {
					stat := deviceStatus(ctx, dev)
					g.Expect(stat.Variant).To(Equal(status.VariantWarning))
					g.Expect(stat.Description).To(Equal("connection reset"))
				}).Should(Succeed())
			},
		)

		It("Should write a status only when the health changes", func(ctx SpecContext) {
			dev := createDevice(ctx, deviceMake)
			st := open(5 * time.Millisecond)
			Expect(st.Start(ctx, driver.NoCommand)).To(Succeed())
			Eventually(func(g Gomega) {
				g.Expect(deviceStatus(ctx, dev).Message).To(Equal("Broker connected"))
			}).Should(Succeed())
			first := deviceStatus(ctx, dev).Time
			checks := scanner.checkCount(dev.Key)
			Eventually(func() int { return scanner.checkCount(dev.Key) }).
				Should(BeNumerically(">", checks+2))
			Expect(deviceStatus(ctx, dev).Time).To(Equal(first))
		})

		It(
			"Should restore a status that a stale write replaced",
			func(ctx SpecContext) {
				dev := createDevice(ctx, deviceMake)
				st := open(5 * time.Millisecond)
				Expect(st.Start(ctx, driver.NoCommand)).To(Succeed())
				Eventually(func(g Gomega) {
					g.Expect(deviceStatus(ctx, dev).Message).
						To(Equal("Broker connected"))
				}).Should(Succeed())
				stale := deviceStatus(ctx, dev)
				stale.Variant = status.VariantWarning
				stale.Message = "Broker state unknown"
				Expect(statusSvc.NewWriter(nil).Set(ctx, &stale)).To(Succeed())
				Eventually(func(g Gomega) {
					stat := deviceStatus(ctx, dev)
					g.Expect(stat.Variant).To(Equal(status.VariantSuccess))
					g.Expect(stat.Message).To(Equal("Broker connected"))
				}).Should(Succeed())
			},
		)

		It("Should give a status to a device that is deleted and created again",
			func(ctx SpecContext) {
				dev := createDevice(ctx, deviceMake)
				witness := createDevice(ctx, deviceMake)
				st := open(5 * time.Millisecond)
				Expect(st.Start(ctx, driver.NoCommand)).To(Succeed())
				Eventually(func(g Gomega) {
					g.Expect(deviceStatus(ctx, dev).Message).
						To(Equal("Broker connected"))
				}).Should(Succeed())
				writeDevices(ctx, func(w device.Writer) error {
					return w.Delete(ctx, dev.Key)
				})
				Expect(deviceStatus(ctx, dev).Message).To(BeEmpty())
				// The second check of witness belongs to a scan that began after the
				// delete, so that scan forgets dev.
				checks := scanner.checkCount(witness.Key)
				Eventually(func() int { return scanner.checkCount(witness.Key) }).
					Should(BeNumerically(">", checks+1))
				writeDevices(ctx, func(w device.Writer) error {
					return w.Create(ctx, &dev)
				})
				Eventually(func(g Gomega) {
					g.Expect(deviceStatus(ctx, dev).Message).
						To(Equal("Broker connected"))
				}).Should(Succeed())
			},
		)

		It("Should warn with the error of a scan that fails", func(ctx SpecContext) {
			dev := createDevice(ctx, deviceMake)
			st := openOn(
				gorp.Wrap(refusingDB{DB: db.KV(), marker: []byte(dev.Key)}),
				time.Hour,
			)
			Expect(st.Start(ctx, driver.NoCommand)).To(Succeed())
			Eventually(func(g Gomega) {
				stat := taskStatus(ctx)
				g.Expect(stat.Variant).To(Equal(status.VariantWarning))
				g.Expect(stat.Message).To(Equal("Failed to scan for devices"))
				g.Expect(stat.Description).To(Equal("commit failed"))
				g.Expect(stat.Details.Running).To(BeTrue())
			}).Should(Succeed())
			Expect(deviceStatus(ctx, dev).Message).ToNot(Equal("Broker connected"))
		})

		It("Should stop the checks on a stop command", func(ctx SpecContext) {
			dev := createDevice(ctx, deviceMake)
			st := open(5 * time.Millisecond)
			Expect(st.Start(ctx, driver.NoCommand)).To(Succeed())
			Eventually(func() int { return scanner.checkCount(dev.Key) }).
				Should(BeNumerically(">", 0))
			Expect(st.Exec(ctx, task.Command{Type: "stop", Key: "cmd-1"})).
				To(Succeed())
			checks := scanner.checkCount(dev.Key)
			Consistently(
				func() int { return scanner.checkCount(dev.Key) },
				50*time.Millisecond,
			).
				Should(Equal(checks))
			stat := taskStatus(ctx)
			Expect(stat.Details.Running).To(BeFalse())
			Expect(stat.Details.Cmd).To(Equal("cmd-1"))
		})
	})

	Describe("Exec", func() {
		It("Should answer a scanner command with its data", func(ctx SpecContext) {
			st := open(time.Hour)
			Expect(st.Start(ctx, driver.NoCommand)).To(Succeed())
			Expect(st.Exec(ctx, task.Command{Type: "test_connection", Key: "cmd-2"})).
				To(Succeed())
			Eventually(func(g Gomega) {
				stat := taskStatus(ctx)
				g.Expect(stat.Variant).To(Equal(status.VariantSuccess))
				g.Expect(stat.Message).To(Equal("Command executed successfully"))
				g.Expect(stat.Details.Cmd).To(Equal("cmd-2"))
				g.Expect(stat.Details.Running).To(BeTrue())
				g.Expect(stat.Details.Data).To(HaveKeyWithValue("latency_ms", 4.0))
			}).Should(Succeed())
		})

		It("Should answer a failed scanner command with its error",
			func(ctx SpecContext) {
				st := open(time.Hour)
				Expect(st.Exec(ctx, task.Command{Type: "browse", Key: "cmd-3"})).
					To(Succeed())
				Eventually(func(g Gomega) {
					stat := taskStatus(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Message).To(Equal("broker refused the subscription"))
					g.Expect(stat.Details.Cmd).To(Equal("cmd-3"))
				}).Should(Succeed())
			},
		)

		It("Should not answer a command the scanner does not support",
			func(ctx SpecContext) {
				var writes atomic.Int32
				key := t.OntologyID().String()
				DeferCleanup(statusSvc.Observe().OnChange(func(
					_ context.Context,
					r gorp.TxReader[status.Key, status.Status[any]],
				) {
					for ch := range r {
						if ch.Key == key {
							writes.Add(1)
						}
					}
				}))
				st := open(time.Hour)
				Expect(st.Exec(ctx, task.Command{Type: "tare", Key: "cmd-4"})).
					To(Succeed())
				// Commands run one at a time, so the answer to cmd-5 comes after cmd-4
				// is done.
				Expect(st.Exec(ctx, task.Command{
					Type: "test_connection", Key: "cmd-5",
				})).To(Succeed())
				Eventually(func() string { return taskStatus(ctx).Details.Cmd }).
					Should(Equal("cmd-5"))
				Expect(writes.Load()).To(BeEquivalentTo(1))
			},
		)

		It("Should return before the scanner command does", func(ctx SpecContext) {
			st := open(time.Hour)
			Expect(st.Exec(ctx, task.Command{Type: "wait", Key: "cmd-6"})).
				To(Succeed())
			Eventually(scanner.waiting.Load).Should(BeEquivalentTo(1))
			Eventually(scanner.release).Should(BeSent(struct{}{}))
			Eventually(func(g Gomega) {
				stat := taskStatus(ctx)
				g.Expect(stat.Details.Cmd).To(Equal("cmd-6"))
				g.Expect(stat.Details.Data).To(HaveKeyWithValue("waited", true))
			}).Should(Succeed())
		})

		It("Should run one scanner command at a time", func(ctx SpecContext) {
			st := open(time.Hour)
			Expect(st.Exec(ctx, task.Command{Type: "wait", Key: "cmd-7"})).
				To(Succeed())
			Expect(st.Exec(ctx, task.Command{Type: "wait", Key: "cmd-8"})).
				To(Succeed())
			Eventually(scanner.waiting.Load).Should(BeEquivalentTo(1))
			Consistently(scanner.waiting.Load, 50*time.Millisecond).
				Should(BeEquivalentTo(1))
			Eventually(scanner.release).Should(BeSent(struct{}{}))
			Eventually(scanner.release).Should(BeSent(struct{}{}))
			Eventually(func() string { return taskStatus(ctx).Details.Cmd }).
				Should(Equal("cmd-8"))
			Expect(scanner.peak.Load()).To(BeEquivalentTo(1))
		})

		It("Should end a scanner command at the time limit of its context",
			func(ctx SpecContext) {
				st := open(time.Hour)
				limited, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
				defer cancel()
				Expect(st.Exec(limited, task.Command{Type: "wait", Key: "cmd-9"})).
					To(Succeed())
				Eventually(func(g Gomega) {
					stat := taskStatus(ctx)
					g.Expect(stat.Variant).To(Equal(status.VariantError))
					g.Expect(stat.Message).To(Equal("context deadline exceeded"))
					g.Expect(stat.Details.Cmd).To(Equal("cmd-9"))
				}).Should(Succeed())
			},
		)
	})

	Describe("Stop", func() {
		It("Should cancel the scanner commands and wait for them, with no answer",
			func(ctx SpecContext) {
				st := open(time.Hour)
				Expect(st.Exec(ctx, task.Command{Type: "wait", Key: "cmd-10"})).
					To(Succeed())
				Expect(st.Exec(ctx, task.Command{Type: "wait", Key: "cmd-11"})).
					To(Succeed())
				Eventually(scanner.waiting.Load).Should(BeEquivalentTo(1))
				Expect(st.Stop(false)).To(Succeed())
				Expect(scanner.waiting.Load()).To(BeZero())
				Expect(taskStatus(ctx).Details.Cmd).To(BeEmpty())
			},
		)

		It("Should answer scanner commands sent after a stop", func(ctx SpecContext) {
			st := open(time.Hour)
			Expect(st.Exec(ctx, task.Command{Type: "wait", Key: "cmd-12"})).
				To(Succeed())
			Eventually(scanner.waiting.Load).Should(BeEquivalentTo(1))
			Expect(st.Stop(false)).To(Succeed())
			Expect(st.Exec(ctx, task.Command{Type: "test_connection", Key: "cmd-13"})).
				To(Succeed())
			Eventually(func() string { return taskStatus(ctx).Details.Cmd }).
				Should(Equal("cmd-13"))
		})
	})
})
