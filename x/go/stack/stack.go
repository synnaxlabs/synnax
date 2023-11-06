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
)

// EmptyStackError is an error returned when popping from an empty stack
var EmptyStackError = errors.New("stack is empty")

// Stack is a container datatype that follows the LIFO (Last In First Out) principle
type Stack[T any] struct {
	stack []T
}

// Push adds an element to the top of the stack
func (s *Stack[T]) Push(i T) {
	s.stack = append(s.stack, i)
}

// Pop removes an element from the top of the stack, returns the element and an error if the stack is empty
func (s *Stack[T]) Pop() (val T, err error) {
	if len(s.stack) == 0 {
		return val, EmptyStackError
	}
	i := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return i, nil
}

// Peek returns a pointer to the element at the top of the stack without removing it, returns nil if the stack is empty
func (s *Stack[T]) Peek() *T {
	if len(s.stack) == 0 {
		return nil
	}
	return &s.stack[len(s.stack)-1]
}

// Len returns the number of elements in the stack
func (s *Stack[T]) Len() int {
	return len(s.stack)
}

// Empty returns true if the stack is empty
func (s *Stack[T]) Empty() bool {
	return len(s.stack) == 0
}
