// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

import "io"

type ParametrizeConfig[V ParametrizeVars] interface {
	Next() (V, error)
}

type ParametrizeVars any

func NewParametrize[V ParametrizeVars](config ParametrizeConfig[V]) *Parametrize[V] {
	return &Parametrize[V]{config: config}
}

type Parametrize[V ParametrizeVars] struct {
	config   ParametrizeConfig[V]
	template func(i int, vars V)
}

func (p *Parametrize[V]) Template(template func(i int, vars V)) {
	p.template = template
}

func (p *Parametrize[V]) Construct() {
	i := 0
	for {
		v, err := p.config.Next()
		if err != nil {
			break
		}
		p.template(i, v)
		i++
	}
}

type iterVars[T ParametrizeVars] struct {
	i    int
	vars []T
}

func (iv *iterVars[T]) Next() (T, error) {
	if iv.i >= len(iv.vars) {
		return *new(T), io.EOF
	}
	v := iv.vars[iv.i]
	iv.i++
	return v, nil
}

func IterVars[V ParametrizeVars](vars []V) ParametrizeConfig[V] {
	return &iterVars[V]{vars: vars}
}
