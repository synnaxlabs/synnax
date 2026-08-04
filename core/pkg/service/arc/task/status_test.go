// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/control"
)

var _ = Describe("Status composer", func() {
	newImpl := func() *impl { return &impl{task: task.Task{Name: "test-task"}} }
	conflicts := []controlConflict{{
		channel: channel.Channel{Name: "ch_a"},
		holder:  control.Subject{Name: "operator", Key: "op-1"},
	}}
	runtimeErr := &statusContribution{
		variant:     status.VariantWarning,
		message:     "Runtime error in add",
		description: "boom",
	}

	It("Should show the running baseline when no condition is active", func() {
		t := newImpl()
		t.status.running = true
		stat := t.renderStatus()
		Expect(stat.Variant).To(Equal(status.VariantSuccess))
		Expect(stat.Message).To(Equal("Task started successfully"))
		Expect(stat.Details.Running).To(BeTrue())
	})

	It("Should show a control conflict over the running baseline", func() {
		t := newImpl()
		t.status.running = true
		t.status.conflicts = conflicts
		stat := t.renderStatus()
		Expect(stat.Variant).To(Equal(status.VariantWarning))
		Expect(stat.Description).To(ContainSubstring("ch_a"))
		Expect(stat.Description).To(ContainSubstring("operator"))
	})

	It("Should prefer a runtime error over a control conflict", func() {
		t := newImpl()
		t.status.running = true
		t.status.conflicts = conflicts
		t.status.runtimeErr = runtimeErr
		stat := t.renderStatus()
		Expect(stat.Message).To(Equal("Runtime error in add"))
		Expect(stat.Description).To(Equal("boom"))
	})

	It("Should keep the runtime error when the conflict clears", func() {
		t := newImpl()
		t.status.running = true
		t.status.runtimeErr = runtimeErr
		t.status.conflicts = conflicts
		t.status.conflicts = nil
		stat := t.renderStatus()
		Expect(stat.Message).To(Equal("Runtime error in add"))
	})

	It("Should surface a conflict once a recovered runtime error clears", func() {
		t := newImpl()
		t.status.running = true
		t.status.runtimeErr = runtimeErr
		t.status.conflicts = conflicts
		t.status.runtimeErr = nil
		stat := t.renderStatus()
		Expect(stat.Variant).To(Equal(status.VariantWarning))
		Expect(stat.Description).To(ContainSubstring("ch_a"))
	})

	It("Should not let a reported status hide an active conflict", func() {
		t := newImpl()
		t.status.running = true
		t.status.conflicts = conflicts
		t.status.reported = &statusContribution{variant: status.VariantSuccess, message: "[task] info"}
		stat := t.renderStatus()
		Expect(stat.Variant).To(Equal(status.VariantWarning))
		Expect(stat.Description).To(ContainSubstring("ch_a"))
	})

	It("Should show terminal status and force running false", func() {
		t := newImpl()
		t.status.running = true
		t.status.conflicts = conflicts
		t.status.terminal = &statusContribution{variant: status.VariantSuccess, message: "Task stopped successfully"}
		stat := t.renderStatus()
		Expect(stat.Message).To(Equal("Task stopped successfully"))
		Expect(stat.Details.Running).To(BeFalse())
	})

	It("Should keep the first terminal status so a later stop cannot mask a failure", func() {
		t := newImpl()
		t.status.running = true
		t.status.setTerminal(status.VariantError, "writer failed")
		t.status.setTerminal(status.VariantSuccess, "Task stopped successfully")
		stat := t.renderStatus()
		Expect(stat.Variant).To(Equal(status.VariantError))
		Expect(stat.Message).To(Equal("writer failed"))
	})

	It("Should treat two renders of the same state as equal ignoring time", func() {
		t := newImpl()
		t.status.running = true
		t.status.conflicts = conflicts
		Expect(statusEqual(t.renderStatus(), t.renderStatus())).To(BeTrue())
	})
})
