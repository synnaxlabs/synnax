// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, box, direction, location, xy } from "@synnaxlabs/x";

export type Location = location.Outer | Partial<location.XY> | location.XY;

export interface Preference {
  targetCorner?: Location;
  dialogCorner?: Location;
}

// DialogLocationPreference can be either:
// 1. A Location (treated as target corner preference)
// 2. A paired preference (target and dialog corners)
export type LocationPreference = Location | Preference;

export interface PositionArgs {
  container: box.Crude;
  target: box.Crude;
  dialog: box.Crude;
  offset?: xy.Crude;
  initial?: LocationPreference;
  prefer?: LocationPreference | LocationPreference[];
  disable?: LocationPreference | LocationPreference[];
}

export const parseLocationOptions = (initial?: Location): Partial<location.XY> => {
  if (initial == null) return { x: undefined, y: undefined };
  if (typeof initial !== "string") return initial;
  if (direction.isX(initial)) return { x: initial, y: undefined };
  return { x: undefined, y: initial };
};

export interface PositionReturn {
  targetCorner: location.XY;
  dialogCorner: location.XY;
  adjustedDialog: box.Box;
}

const normalizePreference = (pref: LocationPreference): Preference => {
  if (typeof pref === "string" || "x" in pref || "y" in pref)
    return { targetCorner: pref, dialogCorner: undefined };
  // The only remaining case here is an empty object, which is valid as both a preference
  // and a location.
  return pref as Preference;
};

const buildOptions = ({
  initial,
  prefer = [],
  disable = [],
}: Pick<PositionArgs, "initial" | "prefer" | "disable">): Option[] => {
  const preferences = array.toArray(prefer).map(normalizePreference);
  const disabled = array.toArray(disable).map(normalizePreference);

  // Start with initial preference if provided
  const options: Option[] = [];

  // Add initial preference first if provided and not disabled
  if (initial != null) {
    const normalizedInitial = normalizePreference(initial);
    const targetLoc = parseLocationOptions(normalizedInitial.targetCorner);
    const dialogLoc = parseLocationOptions(normalizedInitial.dialogCorner);

    // Find all XY locations that match the initial preference
    location.XY_LOCATIONS.forEach((t) => {
      if (location.xyEquals(t, location.CENTER)) return;
      if (targetLoc.x != null && t.x !== targetLoc.x) return;
      if (targetLoc.y != null && t.y !== targetLoc.y) return;

      location.XY_LOCATIONS.forEach((d) => {
        if (location.xyEquals(d, location.CENTER)) return;
        if (dialogLoc.x != null && d.x !== dialogLoc.x) return;
        if (dialogLoc.y != null && d.y !== dialogLoc.y) return;

        const opt = { targetCorner: t, dialogCorner: d };
        options.push(opt);
      });
    });
    const first = options.filter((o) => !isDisabled(o, disabled));
    if (first.length == 0) return options.slice(0, 1);
    return first;
  }

  // Add explicit preferences in order
  preferences.forEach((pref) => {
    const targetLoc = parseLocationOptions(pref.targetCorner);
    const dialogLoc = parseLocationOptions(pref.dialogCorner);

    location.XY_LOCATIONS.forEach((t) => {
      if (location.xyEquals(t, location.CENTER)) return;
      if (!location.xyMatches(t, targetLoc)) return;

      location.XY_LOCATIONS.forEach((d) => {
        if (location.xyEquals(d, location.CENTER)) return;
        if (!location.xyMatches(d, dialogLoc)) return;

        const opt = { targetCorner: t, dialogCorner: d };
        if (
          !isDisabled(opt, disabled) &&
          !options.some(
            (o) =>
              location.xyEquals(o.targetCorner, t) &&
              location.xyEquals(o.dialogCorner, d),
          )
        )
          options.push(opt);
      });
    });
  });

  // Add all remaining valid combinations as fallbacks
  location.XY_LOCATIONS.forEach((t) => {
    if (location.xyEquals(t, location.CENTER)) return;

    location.XY_LOCATIONS.forEach((d) => {
      if (location.xyEquals(d, location.CENTER)) return;

      const opt = { targetCorner: t, dialogCorner: d };
      if (
        !isDisabled(opt, disabled) &&
        !options.some(
          (o) =>
            location.xyEquals(o.targetCorner, t) &&
            location.xyEquals(o.dialogCorner, d),
        )
      )
        options.push(opt);
    });
  });

  return options;
};

const isDisabled = (opt: Option, disabled: Preference[]): boolean =>
  disabled.some((d) => {
    const targetLoc = parseLocationOptions(d.targetCorner);
    const dialogLoc = parseLocationOptions(d.dialogCorner);
    // If dialogCorner is CENTER (from normalization), only match on target corner
    if (location.xyEquals(d.dialogCorner as location.XY, location.CENTER))
      return location.xyMatches(opt.targetCorner, targetLoc);

    return (
      location.xyMatches(opt.targetCorner, targetLoc) &&
      location.xyMatches(opt.dialogCorner, dialogLoc)
    );
  });

interface Option {
  targetCorner: location.XY;
  dialogCorner: location.XY;
}

export const position = ({
  container: containerCrude,
  target: targetCrude,
  dialog: dialogCrude,
  initial,
  prefer,
  disable,
  offset,
}: PositionArgs): PositionReturn => {
  const options = buildOptions({ initial, prefer, disable });
  const containerBox = box.construct(containerCrude);
  const targetBox = box.construct(targetCrude);
  const dialogBox = box.construct(dialogCrude);

  const res = options.reduce(
    (best: EvaluateOptionReturn, opt) => {
      const res = evaluateOption({
        ...opt,
        container: containerBox,
        target: targetBox,
        dialog: dialogBox,
        offset,
      });
      return res.area > best.area ? res : best;
    },
    {
      area: -Infinity,
      targetCorner: location.CENTER,
      dialogCorner: location.CENTER,
      adjustedDialog: dialogBox,
    },
  );

  return {
    targetCorner: res.targetCorner,
    dialogCorner: res.dialogCorner,
    adjustedDialog: res.adjustedDialog,
  };
};

interface EvaluateOptionProps extends Option {
  container: box.Box;
  target: box.Box;
  dialog: box.Box;
  offset?: xy.Crude;
}

interface EvaluateOptionReturn extends PositionReturn {
  area: number;
}

const evaluateOption = ({
  targetCorner,
  dialogCorner,
  container,
  target,
  dialog,
  offset,
}: EvaluateOptionProps): EvaluateOptionReturn => {
  let targetPoint = box.xyLoc(target, targetCorner);
  if (offset != null) {
    const parsedOffset = xy.construct(offset);
    if (targetCorner.x === dialogCorner.x) parsedOffset.x = 0;
    if (targetCorner.y === dialogCorner.y) parsedOffset.y = 0;
    targetPoint = xy.translate(targetPoint, targetCorner, parsedOffset);
  }
  const dialogBox = box.constructWithAlternateRoot(
    targetPoint.x,
    targetPoint.y,
    box.width(dialog),
    box.height(dialog),
    dialogCorner,
    location.TOP_LEFT,
  );
  let area = box.area(box.intersection(dialogBox, container));
  if (box.area(box.intersection(target, dialogBox)) !== 0) area = 0;
  return {
    targetCorner,
    dialogCorner,
    adjustedDialog: dialogBox,
    area,
  };
};
