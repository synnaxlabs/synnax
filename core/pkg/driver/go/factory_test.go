// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package godriver_test

import (
	"context"

	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	godriver "github.com/synnaxlabs/synnax/pkg/driver/go"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	. "github.com/synnaxlabs/x/testutil"
)

type mockFactory struct {
	configureFunc func(t task.Task) (godriver.Task, bool, error)
	name          string
}

func (f *mockFactory) ConfigureTask(_ godriver.Context, t task.Task) (godriver.Task, bool, error) {
	if f.configureFunc != nil {
		return f.configureFunc(t)
	}
	return nil, false, nil
}

func (f *mockFactory) Name() string { return f.name }

type mockTask struct {
	execFunc func(cmd task.Command) error
	stopFunc func() error
	key      task.Key
}

func (t *mockTask) Exec(_ context.Context, cmd task.Command) error {
	if t.execFunc != nil {
		return t.execFunc(cmd)
	}
	return nil
}

func (t *mockTask) Stop(_ context.Context, _ bool) error {
	if t.stopFunc != nil {
		return t.stopFunc()
	}
	return nil
}

func (t *mockTask) Key() task.Key { return t.key }

var _ = Describe("MultiFactory", func() {
	var driverCtx godriver.Context
	BeforeEach(func() {
		driverCtx = godriver.Context{Context: ctx}
	})
	Describe("ConfigureTask", func() {
		It("should return task from first matching factory", func() {
			expectedTask := &mockTask{key: 1}
			f1 := &mockFactory{
				name: "first",
				configureFunc: func(_ task.Task) (godriver.Task, bool, error) {
					return expectedTask, true, nil
				},
			}
			f2 := &mockFactory{
				name: "second",
				configureFunc: func(_ task.Task) (godriver.Task, bool, error) {
					Fail("second factory should not be called")
					return nil, false, nil
				},
			}
			mf := godriver.MultiFactory{f1, f2}
			result := MustBeOk(MustSucceed2(mf.ConfigureTask(
				driverCtx,
				task.Task{Type: "test"},
			)))
			Expect(result).To(Equal(expectedTask))
		})

		It("should try next factory when first returns not handled", func() {
			expectedTask := &mockTask{key: 2}
			f1 := &mockFactory{
				name: "first",
				configureFunc: func(_ task.Task) (godriver.Task, bool, error) {
					return nil, false, nil
				},
			}
			f2 := &mockFactory{
				name: "second",
				configureFunc: func(_ task.Task) (godriver.Task, bool, error) {
					return expectedTask, true, nil
				},
			}
			mf := godriver.MultiFactory{f1, f2}
			result := MustBeOk(MustSucceed2(mf.ConfigureTask(
				driverCtx,
				task.Task{Type: "test"}),
			))
			Expect(result).To(Equal(expectedTask))
		})

		It("should stop on first error even if later factories might handle", func() {
			expectedErr := errors.New("config error")
			f1 := &mockFactory{
				name: "first",
				configureFunc: func(_ task.Task) (godriver.Task, bool, error) {
					return nil, true, expectedErr
				},
			}
			f2 := &mockFactory{
				name: "second",
				configureFunc: func(_ task.Task) (godriver.Task, bool, error) {
					Fail("second factory should not be called")
					return nil, false, nil
				},
			}
			mf := godriver.MultiFactory{f1, f2}
			t, ok, err := mf.ConfigureTask(godriver.Context{Context: ctx}, task.Task{Type: "test"})
			Expect(t).To(BeNil())
			Expect(err).To(MatchError(expectedErr))
			Expect(ok).To(BeTrue())
		})

		It("should return not handled when no factory matches", func() {
			f1 := &mockFactory{
				name: "first",
				configureFunc: func(_ task.Task) (godriver.Task, bool, error) {
					return nil, false, nil
				},
			}
			f2 := &mockFactory{
				name: "second",
				configureFunc: func(_ task.Task) (godriver.Task, bool, error) {
					return nil, false, nil
				},
			}
			mf := godriver.MultiFactory{f1, f2}
			result, ok := MustSucceed2(mf.ConfigureTask(
				godriver.Context{Context: ctx},
				task.Task{Type: "test"}),
			)
			Expect(ok).To(BeFalse())
			Expect(result).To(BeNil())
		})

		It("should handle empty factory list", func() {
			mf := godriver.MultiFactory{}
			result, ok := MustSucceed2(mf.ConfigureTask(godriver.Context{Context: ctx}, task.Task{Type: "test"}))
			Expect(ok).To(BeFalse())
			Expect(result).To(BeNil())
		})
	})

	Describe("Name", func() {
		It("should return multi", func() {
			mf := godriver.MultiFactory{}
			Expect(mf.Name()).To(Equal("multi"))
		})
	})
})
