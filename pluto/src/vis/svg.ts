// copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { XY, ZERO_XY, Direction, toXY, UnparsedXY } from "@synnaxlabs/x";
import { Dimensions } from "reactflow";

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
  viewBox: (dims: Dimensions, offset: XY = ZERO_XY) =>
    `${offset.x} ${offset.y} ${dims.width} ${dims.height}`,
};
