// Copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { XY, Direction, LooseXYT, LooseDirectionT } from "@synnaxlabs/x";
import { Dimensions } from "reactflow";

export const SVG = {
  translate: (xy: LooseXYT) => {
    const xy_ = new XY(xy);
    return `translate(${xy_.x}, ${xy_.y})`;
  },
  translateIn: (amount: number, dir: LooseDirectionT) =>
    new Direction(dir).equals("x")
      ? `translate(${amount}, 0)`
      : `translate(0, ${amount})`,
  line: (one: LooseXYT, two: LooseXYT) => {
    const one_ = new XY(one);
    const two_ = new XY(two);
    return { x1: one_.x, y1: one_.y, x2: two_.x, y2: two_.y };
  },
  viewBox: (dims: Dimensions, offset: XY = XY.ZERO) =>
    `${offset.x} ${offset.y} ${dims.width} ${dims.height}`,
};
