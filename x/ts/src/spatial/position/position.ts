// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Alignment,
  type Order,
  type XLocation,
  type YLocation,
} from "@/spatial/base";
import { box } from "@/spatial/box";
import { direction } from "@/spatial/direction";
import { location } from "@/spatial/location";
import { xy } from "@/spatial/xy";

export const posititonSoVisible = (target: HTMLElement, p: xy.XY): [xy.XY, boolean] => {
  const { width, height } = target.getBoundingClientRect();
  const { innerWidth, innerHeight } = window;
  let changed = false;
  let nextXY = xy.construct(p);
  if (p.x + width > innerWidth) {
    nextXY = xy.translateX(nextXY, -width);
    changed = true;
  }
  if (p.y + height > innerHeight) {
    nextXY = xy.translateY(nextXY, -height);
    changed = true;
  }
  return [nextXY, changed];
};

export interface DialogProps {
  container: box.Crude;
  target: box.Crude;
  dialog: box.Crude;
  initial?: location.Outer | Partial<location.Outer> | location.XY;
}

const parseInitialPosition = (
  initial?: location.Outer | Partial<location.Outer> | location.XY,
): Partial<location.XY> => {
  if (initial == null) return { x: undefined, y: undefined };
  const parsedXYLoc = location.xy.safeParse(initial);
  if (parsedXYLoc.success) return parsedXYLoc.data;
  const parsedLoc = location.location.safeParse(initial);
  if (parsedLoc.success) {
    const isX = direction.construct(parsedLoc.data) === "x";
    return isX
      ? { x: parsedLoc.data as XLocation, y: undefined }
      : { x: undefined, y: parsedLoc.data as YLocation };
  }
  throw new Error(`Invalid initial position: ${initial}`);
};

const Y_LOCATION_PREFERENCES: location.Y[] = ["top", "bottom"];
const X_LOCATION_PREFERENCES: location.X[] = ["left", "right"];
const OUTER_LOCATION_PREFERENCES: location.Outer[] = [
  ...X_LOCATION_PREFERENCES,
  ...Y_LOCATION_PREFERENCES,
];

interface BestLocationProps<C extends location.Location> {
  target: box.Crude;
  dialog: box.Crude;
  container: box.Crude;
  options: C[];
  direction: direction.Direction;
}

const bestLocation = <C extends location.Location>({
  target,
  dialog,
  container,
  options,
  direction,
}: BestLocationProps<C>): C =>
  options.find((l) => {
    const distance = Math.abs(box.loc(container, l) - box.loc(target, l));
    return distance > box.dim(dialog, direction);
  }) ?? options[0];

export const dialog = ({
  container,
  target,
  dialog,
  initial,
}: DialogProps): Partial<location.XY> => {
  const initialPos = parseInitialPosition(initial);
  const options = location.XY_LOCATIONS.filter(
    (l) =>
      !location.xyEquals(l, location.CENTER) &&
      (initialPos.x == null || l.x === initialPos.x) &&
      (initialPos.y == null || l.y === initialPos.y),
  );

  return initialPos;
};

interface EvaluateOptionProps {
  option: location.XY;
  order: Alignment;
  container: box.Crude;
  target: box.Crude;
  dialog: box.Crude;
}

const evaluateOption = ({
  option,
  order,
  container,
  target,
  dialog,
}: EvaluateOptionProps): number => {
  const root = getRoot(option, order);
};

// REASONING TABLE
//
// TL, S -> BR
// TL, C -> BC
// TL, E -> BL
// TR, S -> BL
// TR, C -> BC
// TR, E -> BR
// TC, S -> BL
// TC, C -> BC
// TC, E -> BR
// CL, S -> BR
// CL, C -> CR
// CL, E -> TR
// CR, S -> BL
// CR, C -> CL
// CR, E -> TL
// BL, S -> TR
// BL, C -> TC
// BL, E -> TL
// BR, S -> TL
// BR, C -> TC
// BR, E -> TR
// BC, S -> TL
// BC, C -> TC
// BC, E -> TR
// BL, S -> TR
// BL, C -> TC
// BL, E -> TL

const X_ALIGNMENT_MAP: Record<Alignment, location.X | location.Center> = {
  start: "left",
  center: "center",
  end: "right",
};

const Y_ALIGNMENT_MAP: Record<Alignment, location.Y | location.Center> = {
  start: "top",
  center: "center",
  end: "bottom",
};

export const getRoot = (option: location.XY, order: Alignment): location.XY => {
  const out: location.XY = { x: "center", y: "center" };
  if (option.y !== "center") {
    out.y = location.swap(option.y) as location.Y;
    const swapper = option.y === "bottom" ? location.swap : (v: location.Location) => v;
    out.x = swapper(X_ALIGNMENT_MAP[order]) as location.X;
  } else {
    out.x = location.swap(option.x) as location.X;
    out.y = Y_ALIGNMENT_MAP[order];
  }
  return out;
};
