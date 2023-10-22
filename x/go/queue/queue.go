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
	"errors"
)

type Queue[T any] struct {
	queue []T
}

// Add an element to the end of the queue
func (q *Queue[T]) Push(i T) {
	q.queue = append(q.queue, i)
}

// Remove an element from the front of the queue, returns the element or an error if the queue is empty
func (q *Queue[T]) Pop() (val T, err error) {
	if len(q.queue) == 0 {
		return val, errors.New("queue is empty")
	}
	i := q.queue[0]
	q.queue = q.queue[1:]
	return i, nil
}

// Returns a pointer to the element at the front of the queue without removing it, returns nil if the queue is empty
func (q *Queue[T]) Peek() *T {
	if len(q.queue) == 0 {
		return nil
	}
	return &q.queue[0]
}

// Returns the number of elements in the queue
func (q *Queue[T]) Len() int {
	return len(q.queue)
}

// Returns true if the queue is empty
func (q *Queue[T]) Empty() bool {
	return len(q.queue) == 0
}
