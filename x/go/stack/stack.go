// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stack

import (
	"errors"
)

type Stack[T any] struct {
	stack []T
}

func (s *Stack[T]) Push(i T) {
	s.stack = append(s.stack, i)
}

func (s *Stack[T]) Pop() (val T, err error) {
	if len(s.stack) == 0 {
		return val, errors.New("stack is empty")
	}
	i := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return i, nil
}

func (s *Stack[T]) Peek() *T {
	if len(s.stack) == 0 {
		return nil
	}
	return &s.stack[len(s.stack)-1]
}

func (s *Stack[T]) Len() int {
	return len(s.stack)
}

func (s *Stack[T]) Empty() bool {
	return len(s.stack) == 0
}
