// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stack

type Stack struct {
	stack []interface{}
}

func (s *Stack) Push(i interface{}) {
	s.stack = append(s.stack, i)
}

func (s *Stack) Pop() interface{} {
	if len(s.stack) == 0 {
		return nil
	}
	i := s.stack[len(s.stack)-1]
	s.stack = s.stack[:len(s.stack)-1]
	return i
}

func (s *Stack) Peek() *interface{} {
	if len(s.stack) == 0 {
		return nil
	}
	return &s.stack[len(s.stack)-1]
}

func (s *Stack) Len() int {
	return len(s.stack)
}

func (s *Stack) Empty() bool {
	return len(s.stack) == 0
}
