// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bounds, TimeSpan, TimeStamp } from "@synnaxlabs/x";

import { TickType } from "../../Axis/TickFactory";

const AXIS_SIZE_UPADTE_UPPER_THRESHOLD = 2; // px;
const AXIS_SIZE_UPDATE_LOWER_THRESHOLD = 7;

export const withinSizeThreshold = (prev: number, next: number): boolean =>
  new Bounds({
    lower: prev - AXIS_SIZE_UPDATE_LOWER_THRESHOLD,
    upper: prev + AXIS_SIZE_UPADTE_UPPER_THRESHOLD,
  }).contains(next);

const EMPTY_LINEAR_BOUNDS = new Bounds({ lower: 0, upper: 1 });
const now = TimeStamp.now();
const EMPTY_TIME_BOUNDS = new Bounds({
  lower: now.valueOf(),
  upper: now.add(TimeSpan.HOUR).valueOf(),
});

export const autoBounds = (
  bounds: Bounds[],
  padding: number = 0.1,
  type: TickType
): [Bounds, number] => {
  if (bounds.length === 0) {
    if (type === "linear") return [EMPTY_LINEAR_BOUNDS, 0];
    return [EMPTY_TIME_BOUNDS, 0];
  }
  const { upper, lower } = Bounds.max(bounds);
  if (upper === lower)
    return [new Bounds({ lower: lower - 1, upper: upper + 1 }), lower];
  const _padding = (upper - lower) * padding;
  return [new Bounds({ lower: lower - _padding, upper: upper + _padding }), lower];
};
