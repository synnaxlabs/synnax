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
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/deque"
)

// EmptyStackError is an error returned when popping from an empty stack
var EmptyStackError = errors.New("stack is empty")

// Stack is a container datatype that follows the LIFO (Last In First Out) principle
type Stack[T any] struct {
	s deque.Deque[T]
}

// Push adds an element to the top of the stack
func (s *Stack[T]) Push(i T) {
	s.s.PushBack(i)
}

// Pop removes an element from the top of the stack, returns the element and an error if the stack is empty
func (s *Stack[T]) Pop() (val T, err error) {
	if s.Len() == 0 {
		return val, EmptyStackError
	}
	return s.s.PopBack(), nil
}

// Peek returns a copy of the element at the top of the stack without removing it
// Panics if the stack is empty
func (s *Stack[T]) Peek() (val T) {
	return s.s.Back()
}

// Len returns the number of elements in the stack
func (s *Stack[T]) Len() int {
	return s.s.Len()
}

// Empty returns true if the stack is empty
func (s *Stack[T]) Empty() bool {
	return s.Len() == 0
}

// Cap returns the capacity of the stack
func (s *Stack[T]) Cap() int {
	return s.s.Cap()
}
