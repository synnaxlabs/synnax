// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bounds } from "@synnaxlabs/x";

export const autoBounds = (
  bounds: Bounds[],
  padding: number = 0.1
): [Bounds, number] => {
  if (bounds.length === 0) return [new Bounds({ lower: 0, upper: 1 }), 0];
  const { upper, lower } = Bounds.max(bounds);
  if (upper === lower)
    return [new Bounds({ lower: lower - 1, upper: upper - 1 }), lower];
  const _padding = (upper - lower) * padding;
  return [new Bounds({ lower: lower - _padding, upper: upper + _padding }), lower];
};
