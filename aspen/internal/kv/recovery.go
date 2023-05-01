// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"github.com/synnaxlabs/x/confluence"
	kvx "github.com/synnaxlabs/x/kv"
)

type recoveryTransform struct {
	Config
	confluence.LinearTransform[TxRequest, TxRequest]
	repetitions map[string]int
}

func newRecoveryTransform(cfg Config) segment {
	r := &recoveryTransform{Config: cfg, repetitions: make(map[string]int)}
	r.Transform = r.transform
	return r
}

func (r *recoveryTransform) transform(
	_ context.Context,
	in TxRequest,
) (out TxRequest, ok bool, err error) {
	out.Context = in.Context
	for _, op := range in.Operations {
		key, err := kvx.CompositeKey(op.Key, op.Version)
		if err != nil {
			panic(err)
		}
		strKey := string(key)
		if r.repetitions[strKey] > r.RecoveryThreshold {
			op.state = recovered
			out.Operations = append(out.Operations, op)
			delete(r.repetitions, strKey)
		}
		r.repetitions[strKey]++
	}
	return out, true, nil
}
