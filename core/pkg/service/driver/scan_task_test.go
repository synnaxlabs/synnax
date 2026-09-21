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
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

// mapScanner is a driver.Scanner whose check results a spec sets per device.
type mapScanner struct {
	mu sync.Mutex
	// unreachable holds the check error of each device that is not reachable.
	unreachable map[device.Key]error
	checks      map[device.Key]int
}

func newMapScanner() *mapScanner {
	return &mapScanner{
		unreachable: make(map[device.Key]error),
		checks:      make(map[device.Key]int),
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
	_ context.Context,
	cmd task.Command,
) (msgpack.EncodedJSON, error) {
	switch cmd.Type {
	case "test_connection":
		return msgpack.EncodedJSON{"latency_ms": 4.0}, nil
	case "browse":
		return nil, errors.New("broker refused the subscription")
	}
	return nil, driver.ErrUnsupportedCommand
}

var _ = Describe("ScanTask", func() {
	var (
		t       task.Task
		scanner *mapScanner
		rackKey rack.Key
		make_   string
	)

	BeforeEach(func(ctx SpecContext) {
		r := rack.Rack{Name: "scan-task-rack"}
		Expect(rackService.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		rackKey = r.Key
		make_ = "make-" + uuid.NewString()
		t = task.Task{
			Key:  uuid.New(),
			Name: "scan-task-test",
			Type: "mock",
			Rack: rackKey,
		}
		scanner = newMapScanner()
	})

	open := func(interval time.Duration) *driver.ScanTask {
		GinkgoHelper()
		st := MustSucceed(driver.NewScanTask(driver.ScanTaskConfig{
			Scanner:            scanner,
			Status:             statusSvc,
			Device:             deviceSvc,
			Make:               make_,
			ReachableMessage:   "Broker connected",
			UnreachableMessage: "Failed to reach broker",
			Task:               t,
			Interval:           interval,
		}))
		DeferCleanup(func() { Expect(st.Stop(false)).To(Succeed()) })
		return st
	}

	createDevice := func(ctx context.Context, deviceMake string) device.Device {
		GinkgoHelper()
		dev := device.Device{
			Key:      uuid.NewString(),
			Rack:     rackKey,
			Location: "localhost:1883",
			Name:     "Broker",
			Make:     deviceMake,
			Model:    "broker",
		}
		Expect(deviceSvc.NewWriter(nil).Create(ctx, &dev)).To(Succeed())
		return dev
	}

	deviceStatus := func(ctx context.Context, dev device.Device) device.Status {
		var stat device.Status
		_ = statusSvc.NewRetrieve[device.StatusDetails]().
			Where(status.MatchKeys[device.StatusDetails](dev.OntologyID().String())).
			Entry(&stat).Exec(ctx, nil)
		return stat
	}

	taskStatus := func(ctx context.Context) task.Status {
		var stat task.Status
		_ = statusSvc.NewRetrieve[task.StatusDetails]().
			Where(status.MatchKeys[task.StatusDetails](t.OntologyID().String())).
			Entry(&stat).Exec(ctx, nil)
		return stat
	}

	Describe("NewScanTask", func() {
		It("Should reject a configuration with no make", func() {
			Expect(driver.NewScanTask(driver.ScanTaskConfig{
				Scanner:            scanner,
				Status:             statusSvc,
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
				up := createDevice(ctx, make_)
				down := createDevice(ctx, make_)
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
			dev := createDevice(ctx, make_)
			Eventually(func(g Gomega) {
				g.Expect(deviceStatus(ctx, dev).Message).To(Equal("Broker connected"))
			}).Should(Succeed())
		})

		It("Should report a change of health on the next interval",
			func(ctx SpecContext) {
				dev := createDevice(ctx, make_)
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
			dev := createDevice(ctx, make_)
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

		It("Should stop the checks on a stop command", func(ctx SpecContext) {
			dev := createDevice(ctx, make_)
			st := open(5 * time.Millisecond)
			Expect(st.Start(ctx, driver.NoCommand)).To(Succeed())
			Eventually(func() int { return scanner.checkCount(dev.Key) }).
				Should(BeNumerically(">", 0))
			Expect(st.Exec(ctx, task.Command{Type: "stop", Key: "cmd-1"})).
				To(Succeed())
			checks := scanner.checkCount(dev.Key)
			Consistently(func() int { return scanner.checkCount(dev.Key) }, "50ms").
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
			stat := taskStatus(ctx)
			Expect(stat.Variant).To(Equal(status.VariantSuccess))
			Expect(stat.Details.Cmd).To(Equal("cmd-2"))
			Expect(stat.Details.Running).To(BeTrue())
			Expect(stat.Details.Data).To(HaveKeyWithValue("latency_ms", 4.0))
		})

		It("Should answer a failed scanner command with its error",
			func(ctx SpecContext) {
				st := open(time.Hour)
				Expect(st.Exec(ctx, task.Command{Type: "browse", Key: "cmd-3"})).
					To(MatchError(ContainSubstring("broker refused the subscription")))
				stat := taskStatus(ctx)
				Expect(stat.Variant).To(Equal(status.VariantError))
				Expect(stat.Message).To(Equal("broker refused the subscription"))
				Expect(stat.Details.Cmd).To(Equal("cmd-3"))
			},
		)

		It("Should reject a command the scanner does not support",
			func(ctx SpecContext) {
				st := open(time.Hour)
				Expect(st.Exec(ctx, task.Command{Type: "tare", Key: "cmd-4"})).
					To(MatchError(driver.ErrUnsupportedCommand))
				Expect(taskStatus(ctx).Details.Cmd).ToNot(Equal("cmd-4"))
			},
		)
	})
})
