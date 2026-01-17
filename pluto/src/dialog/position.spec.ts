// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, location, testutil } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Dialog } from "@/dialog";

interface Spec {
  args: Dialog.PositionArgs;
  name?: string;
  expected: Dialog.PositionReturn;
}

describe("position", () => {
  describe("dialog", () => {
    const CENTER: Spec = {
      name: "target in center, plenty of space, no preference",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.CENTER_RIGHT,
        adjustedDialog: box.construct(20, 35, 20, 20),
      },
    };

    const TOP_LEFT: Spec = {
      name: "target in top left, plenty of space, no preference",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(0, 0, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
      },
      expected: {
        targetCorner: location.CENTER_RIGHT,
        dialogCorner: location.TOP_LEFT,
        adjustedDialog: box.construct(10, 5, 20, 20),
      },
    };

    const BOTTOM_RIGHT: Spec = {
      name: "target in bottom right, plenty of space, no preference",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(90, 90, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.BOTTOM_RIGHT,
        adjustedDialog: box.construct(70, 75, 20, 20),
      },
    };

    const BOTTOM_LEFT: Spec = {
      name: "target in bottom left, plenty of space, no preference",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(0, 90, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
      },
      expected: {
        targetCorner: location.CENTER_RIGHT,
        dialogCorner: location.BOTTOM_LEFT,
        adjustedDialog: box.construct(10, 75, 20, 20),
      },
    };

    const TOP_RIGHT: Spec = {
      name: "target in top right, plenty of space, no preference",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(90, 0, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.TOP_RIGHT,
        adjustedDialog: box.construct(70, 5, 20, 20),
      },
    };

    const CENTER_LEFT: Spec = {
      name: "target in center, plenty of space, preference for left side",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        prefer: {
          targetCorner: "left",
          dialogCorner: "right",
        },
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.CENTER_RIGHT,
        adjustedDialog: box.construct(20, 35, 20, 20),
      },
    };

    const CENTER_BOTTOM_RIGHT: Spec = {
      name: "target in center, preference for bottom-right/top-left pairing",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        prefer: {
          targetCorner: location.BOTTOM_RIGHT,
          dialogCorner: location.TOP_LEFT,
        },
      },
      expected: {
        targetCorner: location.BOTTOM_RIGHT,
        dialogCorner: location.TOP_LEFT,
        adjustedDialog: box.construct(50, 50, 20, 20),
      },
    };

    // Dialog larger than container
    const TOP_LEFT_LARGE: Spec = {
      name: "box in top left, preference for top-left/bottom-right pairing",
      args: {
        container: box.construct(0, 0, 50, 50),
        target: box.construct(0, 0, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        prefer: {
          targetCorner: location.TOP_LEFT,
          dialogCorner: location.BOTTOM_RIGHT,
        },
      },
      expected: {
        targetCorner: location.CENTER_RIGHT,
        dialogCorner: location.TOP_LEFT,
        adjustedDialog: box.construct(10, 5, 20, 20),
      },
    };

    const MULTIPLE_PREFER: Spec = {
      name: "multiple paired preferences",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        prefer: [
          { targetCorner: location.TOP_LEFT, dialogCorner: location.BOTTOM_RIGHT },
          { targetCorner: location.BOTTOM_RIGHT, dialogCorner: location.TOP_LEFT },
        ],
      },
      expected: {
        targetCorner: location.TOP_LEFT,
        dialogCorner: location.BOTTOM_RIGHT,
        adjustedDialog: box.construct(20, 20, 20, 20),
      },
    };

    const DISABLED_LOCATIONS: Spec = {
      name: "disabled pairings forcing fallback",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        disable: [
          { targetCorner: location.CENTER_LEFT, dialogCorner: location.CENTER_RIGHT },
          { targetCorner: location.CENTER_RIGHT, dialogCorner: location.CENTER_LEFT },
          { targetCorner: location.TOP_CENTER, dialogCorner: location.BOTTOM_CENTER },
        ],
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.TOP_RIGHT,
        adjustedDialog: box.construct(20, 45, 20, 20),
      },
    };

    const PREFER_AND_DISABLE: Spec = {
      name: "preferences with both prefer and disable options",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        prefer: [
          { targetCorner: location.TOP_LEFT, dialogCorner: location.BOTTOM_RIGHT },
          { targetCorner: location.CENTER_LEFT, dialogCorner: location.CENTER_RIGHT },
        ],
        disable: [
          { targetCorner: location.TOP_LEFT, dialogCorner: location.BOTTOM_RIGHT },
        ],
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.CENTER_RIGHT,
        adjustedDialog: box.construct(20, 35, 20, 20),
      },
    };

    const DIALOG_PREFERENCES: Spec = {
      name: "paired preferences for dialog positioning",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        prefer: [
          { targetCorner: location.CENTER_RIGHT, dialogCorner: location.TOP_LEFT },
          { targetCorner: location.CENTER_RIGHT, dialogCorner: location.BOTTOM_LEFT },
        ],
      },
      expected: {
        targetCorner: location.CENTER_RIGHT,
        dialogCorner: location.TOP_LEFT,
        adjustedDialog: box.construct(50, 45, 20, 20),
      },
    };

    const BOTH_PREFERENCES: Spec = {
      name: "specific paired preference",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        prefer: {
          targetCorner: location.BOTTOM_RIGHT,
          dialogCorner: location.TOP_LEFT,
        },
      },
      expected: {
        targetCorner: location.BOTTOM_RIGHT,
        dialogCorner: location.TOP_LEFT,
        adjustedDialog: box.construct(50, 50, 20, 20),
      },
    };

    const CONFLICTING_PREFERENCES: Spec = {
      name: "disabled preferences resolve to best available option",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        prefer: [
          { targetCorner: location.CENTER_LEFT, dialogCorner: location.CENTER_RIGHT },
        ],
        disable: [
          { targetCorner: location.CENTER_LEFT, dialogCorner: location.CENTER_RIGHT },
          { targetCorner: location.CENTER_RIGHT, dialogCorner: location.CENTER_LEFT },
        ],
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.TOP_RIGHT,
        adjustedDialog: box.construct(20, 45, 20, 20),
      },
    };

    const PARTIAL_XY_PREFER: Spec = {
      name: "partial XY preferences (prefer left to right)",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        prefer: {
          targetCorner: { x: "left" },
          dialogCorner: { x: "right" },
        },
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.CENTER_RIGHT,
        adjustedDialog: box.construct(20, 35, 20, 20),
      },
    };

    const PARTIAL_XY_DISABLE: Spec = {
      name: "partial XY disable (disable top target positions)",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        disable: {
          targetCorner: { y: "top" },
          dialogCorner: location.CENTER,
        },
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.CENTER_RIGHT,
        adjustedDialog: box.construct(20, 35, 20, 20),
      },
    };

    const CONSTRAINED_SPACE: Spec = {
      name: "constrained space with preferences forcing suboptimal positioning",
      args: {
        container: box.construct(0, 0, 60, 60),
        target: box.construct(45, 45, 10, 10),
        dialog: box.construct(0, 0, 25, 25),
        prefer: {
          targetCorner: location.BOTTOM_RIGHT,
          dialogCorner: location.TOP_LEFT,
        },
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.BOTTOM_RIGHT,
        adjustedDialog: box.construct(20, 25, 25, 25),
      },
    };

    const EDGE_CASE_SINGLE_AXIS: Spec = {
      name: "single axis preference with disable",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        prefer: {
          targetCorner: "right",
          dialogCorner: "left",
        },
        disable: [
          { targetCorner: location.TOP_RIGHT, dialogCorner: location.TOP_LEFT },
          { targetCorner: location.BOTTOM_RIGHT, dialogCorner: location.BOTTOM_LEFT },
        ],
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.CENTER_RIGHT,
        adjustedDialog: box.construct(20, 35, 20, 20),
      },
    };

    const LARGE_DIALOG: Spec = {
      name: "oversized dialog with preferences and constraints",
      args: {
        container: box.construct(0, 0, 80, 80),
        target: box.construct(30, 30, 10, 10),
        dialog: box.construct(0, 0, 50, 30),
        prefer: {
          targetCorner: location.TOP_LEFT,
          dialogCorner: location.BOTTOM_CENTER,
        },
        disable: [
          { targetCorner: location.CENTER_LEFT, dialogCorner: location.CENTER },
          { targetCorner: location.CENTER_RIGHT, dialogCorner: location.CENTER },
          { targetCorner: location.CENTER, dialogCorner: location.BOTTOM_RIGHT },
          { targetCorner: location.CENTER, dialogCorner: location.BOTTOM_LEFT },
        ],
      },
      expected: {
        targetCorner: location.TOP_LEFT,
        dialogCorner: location.BOTTOM_CENTER,
        adjustedDialog: box.construct(5, 0, 50, 30),
      },
    };

    const INITIAL_LOCATION_SIMPLE: Spec = {
      name: "initial location preference",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        initial: {
          targetCorner: location.TOP_RIGHT,
          dialogCorner: location.BOTTOM_LEFT,
        },
      },
      expected: {
        targetCorner: location.TOP_RIGHT,
        dialogCorner: location.BOTTOM_LEFT,
        adjustedDialog: box.construct(50, 20, 20, 20),
      },
    };

    const INITIAL_LOCATION_PARTIAL: Spec = {
      name: "initial location with partial preference",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        initial: {
          targetCorner: "top",
          dialogCorner: "bottom",
        },
      },
      expected: {
        targetCorner: location.TOP_CENTER,
        dialogCorner: location.BOTTOM_CENTER,
        adjustedDialog: box.construct(35, 20, 20, 20),
      },
    };

    const INITIAL_WITH_PREFER: Spec = {
      name: "initial location with prefer fallback",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 20, 20),
        dialog: box.construct(0, 0, 20, 20),
        initial: {
          targetCorner: location.TOP_LEFT,
          dialogCorner: location.BOTTOM_LEFT,
        },
        prefer: {
          targetCorner: location.CENTER_LEFT,
          dialogCorner: location.CENTER_RIGHT,
        },
      },
      expected: {
        targetCorner: location.TOP_LEFT,
        dialogCorner: location.BOTTOM_LEFT,
        adjustedDialog: box.construct(40, 20, 20, 20),
      },
    };

    const INITIAL_CONSTRAINED: Spec = {
      name: "initial location constrained by container",
      args: {
        container: box.construct(0, 0, 50, 50),
        target: box.construct(35, 35, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        initial: {
          targetCorner: location.BOTTOM_RIGHT,
          dialogCorner: location.TOP_LEFT,
        },
      },
      expected: {
        targetCorner: location.BOTTOM_RIGHT,
        dialogCorner: location.TOP_LEFT,
        adjustedDialog: box.construct(45, 45, 20, 20),
      },
    };

    const INITIAL_WITH_DISABLE: Spec = {
      name: "initial location with disabled options",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        initial: {
          targetCorner: location.CENTER_LEFT,
          dialogCorner: location.CENTER_RIGHT,
        },
        disable: [
          { targetCorner: location.CENTER_LEFT, dialogCorner: location.CENTER_RIGHT },
        ],
      },
      expected: {
        targetCorner: location.CENTER_LEFT,
        dialogCorner: location.CENTER_RIGHT,
        adjustedDialog: box.construct(20, 35, 20, 20),
      },
    };

    const INITIAL_SINGLE_AXIS: Spec = {
      name: "initial location with single axis preference",
      args: {
        container: box.construct(0, 0, 100, 100),
        target: box.construct(40, 40, 10, 10),
        dialog: box.construct(0, 0, 20, 20),
        initial: "right",
      },
      expected: {
        targetCorner: location.CENTER_RIGHT,
        dialogCorner: location.CENTER_LEFT,
        adjustedDialog: box.construct(50, 35, 20, 20),
      },
    };

    const SPECS: Spec[] = [
      CENTER,
      TOP_LEFT,
      BOTTOM_RIGHT,
      BOTTOM_LEFT,
      TOP_RIGHT,
      CENTER_LEFT,
      CENTER_BOTTOM_RIGHT,
      TOP_LEFT_LARGE,
      MULTIPLE_PREFER,
      DISABLED_LOCATIONS,
      PREFER_AND_DISABLE,
      DIALOG_PREFERENCES,
      BOTH_PREFERENCES,
      CONFLICTING_PREFERENCES,
      PARTIAL_XY_PREFER,
      PARTIAL_XY_DISABLE,
      CONSTRAINED_SPACE,
      EDGE_CASE_SINGLE_AXIS,
      LARGE_DIALOG,
      INITIAL_LOCATION_SIMPLE,
      INITIAL_LOCATION_PARTIAL,
      INITIAL_WITH_PREFER,
      INITIAL_CONSTRAINED,
      INITIAL_WITH_DISABLE,
      INITIAL_SINGLE_AXIS,
    ];

    SPECS.forEach(({ name, args, expected }) =>
      it(`should position dialog correctly for ${name}`, () =>
        expect(Dialog.position(args)).toEqual(expected)),
    );
  });

  describe("parseLocationOptions", () => {
    const TESTS: [Dialog.Location, Partial<location.XY>][] = [
      ["left", { x: "left" }],
      [{ x: "left" }, { x: "left" }],
      [
        { x: "left", y: "top" },
        { x: "left", y: "top" },
      ],
      [{}, { x: undefined, y: undefined }],
    ];
    TESTS.forEach(([arg, expected]) => {
      it(`should return ${testutil.toString(expected)} for ${testutil.toString(arg)}`, () => {
        expect(Dialog.parseLocationOptions(arg)).toEqual(expected);
      });
    });
  });
});
