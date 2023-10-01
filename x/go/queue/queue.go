// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package queue

type Queue struct {
	queue []interface{}
}

func (q *Queue) Push(i interface{}) {
	q.queue = append(q.queue, i)
}

func (q *Queue) Pop() interface{} {
	if len(q.queue) == 0 {
		return nil
	}
	i := q.queue[0]
	q.queue = q.queue[1:]
	return i
}

func (q *Queue) Peek() *interface{} {
	if len(q.queue) == 0 {
		return nil
	}
	return &q.queue[0]
}

func (q *Queue) Len() int {
	return len(q.queue)
}

func (q *Queue) Empty() bool {
	return len(q.queue) == 0
}
