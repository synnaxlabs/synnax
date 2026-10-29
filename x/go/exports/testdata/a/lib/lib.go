// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lib

func Used() {}

func Internal() {}

func Tested() {}

func OtherTests() {}

var Dead = 1

func Open(cfg Config) { Internal() }

type Config struct{ Mode Mode }

type Mode int

func (Mode) Error() string { return "mode" }

type Shape interface{ Area() int }

type Square struct{}

func (Square) Area() int { return 1 }

func (Square) Side() int { return 1 }

func useSide() int { return Square{}.Side() }

type Handler[S any] interface {
	Handle(S) (S, error)
}

type Op struct{}

func (Op) Handle(s int) (int, error) { return s, nil }

type Color int

const (
	Red Color = iota
	Green
)

func (Color) String() string { return "color" }

type hidden struct{}

func (hidden) Exported() {}

func CalledByGenerated() {}

type DoExtra interface {
	Do()
	Extra()
}

type Inner struct{}

func (Inner) Do() {}

type Outer struct{ Inner }

func (Outer) Extra() {}

type Box[T any] struct{}

func (Box[T]) Read() {}

const (
	Big, Small = 1, 2
)

type Imports struct{}

func (Imports) Has() bool { return true }
