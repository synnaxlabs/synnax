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

func (q *Queue[T]) Push(i T) {
	q.queue = append(q.queue, i)
}

func (q *Queue[T]) Pop() (val T, err error) {
	if len(q.queue) == 0 {
		return val, errors.New("queue is empty")
	}
	i := q.queue[0]
	q.queue = q.queue[1:]
	return i, nil
}

func (q *Queue[T]) Peek() (*T, error) {
	if len(q.queue) == 0 {
		return nil, errors.New("queue is empty")
	}
	return &q.queue[0], nil
}

func (q *Queue[T]) Len() int {
	return len(q.queue)
}

func (q *Queue[T]) Empty() bool {
	return len(q.queue) == 0
}
