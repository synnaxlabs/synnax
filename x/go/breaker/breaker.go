// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package breaker

import (
	"time"
)

type Breaker struct {
	BaseInterval time.Duration
	Scale        float32
	MaxRetries   int
	currInterval time.Duration
	retryCount   int
}

func (b *Breaker) Wait() bool {
	time.Sleep(b.currInterval)
	b.currInterval = time.Duration(float32(b.currInterval) * b.Scale)
	b.retryCount++
	if b.retryCount > b.MaxRetries {
		return false
	}
	return true
}

func (b *Breaker) Reset() {
	b.currInterval = b.BaseInterval
	b.retryCount = 0
}
