// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Alignment, type XLocation, type YLocation } from "@/spatial/base";
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
  alignments?: Alignment[];
  initial?: location.Outer | Partial<location.XY> | location.XY;
  disable?: Array<location.Location | Partial<location.XY>>;
}

const parseInitialPosition = (
  initial?: location.Outer | Partial<location.XY> | location.XY,
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
  return initial as Partial<location.XY>;
};

export interface DialogReturn {
  location: location.XY;
  adjustedDialog: box.Box;
}

export const dialog = ({
  container: containerCrude,
  target: targetCrude,
  dialog: dialogCrude,
  initial,
  alignments = ["start"],
  disable = [],
}: DialogProps): DialogReturn => {
  const initialPos = parseInitialPosition(initial);
  const options = location.XY_LOCATIONS.filter(
    (l) =>
      !location.xyEquals(l, location.CENTER) &&
      (initialPos.x == null || l.x === initialPos.x) &&
      (initialPos.y == null || l.y === initialPos.y) &&
      !disable.some((d) => location.xyMatches(l, d)),
  )
    .map((l) => alignments?.map((a) => [l, a]))
    .flat() as Array<[location.XY, Alignment]>;

  const container = box.construct(containerCrude);
  const target = box.construct(targetCrude);
  const dialog = box.construct(dialogCrude);

  // maximum value of a number in js
  let bestOptionArea = -Infinity;
  const res: DialogReturn = { location: location.CENTER, adjustedDialog: dialog };
  options.forEach(([option, alignment]) => {
    const [adjustedBox, area] = evaluateOption({
      option,
      alignment,
      container,
      target,
      dialog,
    });
    if (area > bestOptionArea) {
      bestOptionArea = area;
      res.location = option;
      res.adjustedDialog = adjustedBox;
    }
  });

  return res;
};

interface EvaluateOptionProps {
  option: location.XY;
  alignment: Alignment;
  container: box.Box;
  target: box.Box;
  dialog: box.Box;
}

const evaluateOption = ({
  option,
  alignment,
  container,
  target,
  dialog,
}: EvaluateOptionProps): [box.Box, number] => {
  const root = getRoot(option, alignment);
  const targetPoint = box.xyLoc(target, option);
  const dialogBox = box.constructWithAlternateRoot(
    targetPoint.x,
    targetPoint.y,
    box.width(dialog),
    box.height(dialog),
    root,
    location.TOP_LEFT,
  );
  const area = box.area(box.intersect(dialogBox, container));
  return [dialogBox, area];
};

const X_ALIGNMENT_MAP: Record<Alignment, location.X | location.Center> = {
  start: "left",
  center: "center",
  end: "right",
};

const Y_ALIGNMENT_MAP: Record<Alignment, location.Y | location.Center> = {
  start: "bottom",
  center: "center",
  end: "top",
};

export const getRoot = (option: location.XY, alignment: Alignment): location.XY => {
  const out: location.XY = { x: "center", y: "center" };
  if (option.y !== "center") {
    out.y = location.swap(option.y) as location.Y;
    const swapper = option.x === "left" ? location.swap : (v: location.Location) => v;
    out.x = swapper(X_ALIGNMENT_MAP[alignment]) as location.X;
  } else {
    out.x = location.swap(option.x) as location.X;
    out.y = Y_ALIGNMENT_MAP[alignment];
  }
  return out;
};
