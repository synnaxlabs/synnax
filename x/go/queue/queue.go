// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package queue

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/deque"
)

// EmptyQueueError is an error returned when popping from an empty queue
var EmptyQueueError = errors.New("queue is empty")

// Queue is a container datatype that follows the FIFO (First In First Out) principle
type Queue[T any] struct {
	q deque.Deque[T]
}

// Push adds an element to the end of the queue
func (q *Queue[T]) Push(i T) {
	q.q.PushBack(i)
}

// Pop removes an element from the front of the queue, returns the element or an error if the queue is empty
func (q *Queue[T]) Pop() (val T, err error) {
	if q.q.Empty() {
		return val, EmptyQueueError
	}
	return q.q.PopFront(), nil
}

// Peek returns a copy of the element at the front of the queue without removing it
// Panics if the queue is empty
func (q *Queue[T]) Peek() (val T) {
	return q.q.Front()
}

// Len returns the number of elements in the queue
func (q *Queue[T]) Len() int {
	return q.q.Len()
}

// Empty returns true if the queue is empty
func (q *Queue[T]) Empty() bool {
	return q.q.Empty()
}

func (q *Queue[T]) Cap() int {
	return q.q.Cap()
}
