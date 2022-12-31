// Copyright 2022 Synnax Labs, Inc.
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
	confluence.LinearTransform[BatchRequest, BatchRequest]
	repetitions map[string]int
}

func newRecoveryTransform(cfg Config) segment {
	r := &recoveryTransform{Config: cfg, repetitions: make(map[string]int)}
	r.LinearTransform.Transform = r.transform
	return r
}

func (r *recoveryTransform) transform(
	ctx context.Context,
	in BatchRequest,
) (out BatchRequest, ok bool, err error) {
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
