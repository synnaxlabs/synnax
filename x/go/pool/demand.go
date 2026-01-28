// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pool

type Demand int64

func (d *Demand) Increase(amount Demand) { *d += amount }

func (d *Demand) Decrease(amount Demand) { *d -= amount }
