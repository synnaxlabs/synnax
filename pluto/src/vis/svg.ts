// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Direction, toXY, UnparsedXY } from "@synnaxlabs/x";

export const SVG = {
  translate: (xy: UnparsedXY) => {
    const xy_ = toXY(xy);
    return `translate(${xy_.x}, ${xy_.y})`;
  },
  translateIn: (amount: number, dir: Direction) =>
    dir === "x" ? `translate(${amount}, 0)` : `translate(0, ${amount})`,
  line: (one: UnparsedXY, two: UnparsedXY) => {
    const one_ = toXY(one);
    const two_ = toXY(two);
    return { x1: one_.x, y1: one_.y, x2: two_.x, y2: two_.y };
  },
};
