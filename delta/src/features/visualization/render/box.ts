import { XY } from "./line";

export interface Box {
  left: number;
  top: number;
  width: number;
  height: number;
}

export interface PointBox {
  one: XY;
  two: XY;
}

export const calculateBottom = (parent: Box, child: Box): number =>
  parent.height - (child.top - parent.top) - child.height;
