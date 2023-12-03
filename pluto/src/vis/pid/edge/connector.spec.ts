import { box, type xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  type Segment,
  type BuildNewConnectorProps,
  newConnector,
  type NeedToGoAroundSourceProps,
  needToGoAround,
  travelSegments,
  type PrepareNodeProps,
  prepareNode,
  type MoveConnectorProps,
  moveConnector,
} from "./connector";

describe("connector", () => {
  describe("needToGoAroundSource", () => {
    interface spec {
      name: string;
      props: NeedToGoAroundSourceProps;
      expected: boolean;
    }
    const LEFT_TRUE: spec = {
      name: "left yes",
      props: {
        sourcePos: { x: 0, y: 0 },
        targetPos: { x: 10, y: 0 },
        sourceOrientation: "left",
      },
      expected: true,
    };

    const LEFT_FALSE: spec = {
      name: "left no",
      props: {
        sourcePos: { x: 10, y: 0 },
        targetPos: { x: 0, y: 0 },
        sourceOrientation: "left",
      },
      expected: false,
    };

    const LEFT_EQUAL_FALSE: spec = {
      name: "left equal no",
      props: {
        sourcePos: { x: 10, y: 0 },
        targetPos: { x: 0, y: 0 },
        sourceOrientation: "left",
      },
      expected: false,
    };

    const RIGHT_TRUE: spec = {
      name: "right yes",
      props: {
        sourcePos: { x: 0, y: 0 },
        targetPos: { x: -10, y: 0 },
        sourceOrientation: "right",
      },
      expected: true,
    };

    const RIGHT_FALSE: spec = {
      name: "right no",
      props: {
        sourcePos: { x: -10, y: 0 },
        targetPos: { x: 0, y: 0 },
        sourceOrientation: "right",
      },
      expected: false,
    };

    const RIGHT_EQUAL_FALSE: spec = {
      name: "right equal no",
      props: {
        sourcePos: { x: -10, y: 0 },
        targetPos: { x: 0, y: 0 },
        sourceOrientation: "right",
      },
      expected: false,
    };

    const TOP_TRUE: spec = {
      name: "top yes",
      props: {
        sourcePos: { x: 0, y: 0 },
        targetPos: { x: 0, y: 10 },
        sourceOrientation: "top",
      },
      expected: true,
    };

    const TOP_FALSE: spec = {
      name: "top no",
      props: {
        sourcePos: { x: 0, y: 10 },
        targetPos: { x: 0, y: 0 },
        sourceOrientation: "top",
      },
      expected: false,
    };

    const TOP_EQUAL_FALSE: spec = {
      name: "top equal no",
      props: {
        sourcePos: { x: 0, y: 10 },
        targetPos: { x: 0, y: 0 },
        sourceOrientation: "top",
      },
      expected: false,
    };

    const BOTTOM_TRUE: spec = {
      name: "bottom yes",
      props: {
        sourcePos: { x: 0, y: 0 },
        targetPos: { x: 0, y: -10 },
        sourceOrientation: "bottom",
      },
      expected: true,
    };

    const BOTTOM_FALSE: spec = {
      name: "bottom no",
      props: {
        sourcePos: { x: 0, y: -10 },
        targetPos: { x: 0, y: 0 },
        sourceOrientation: "bottom",
      },
      expected: false,
    };

    const BOTTOM_EQUAL_FALSE: spec = {
      name: "bottom equal no",
      props: {
        sourcePos: { x: 0, y: -10 },
        targetPos: { x: 0, y: 0 },
        sourceOrientation: "bottom",
      },
      expected: false,
    };

    const SPECS = [
      LEFT_TRUE,
      LEFT_FALSE,
      LEFT_EQUAL_FALSE,
      RIGHT_TRUE,
      RIGHT_FALSE,
      RIGHT_EQUAL_FALSE,
      TOP_TRUE,
      TOP_FALSE,
      TOP_EQUAL_FALSE,
      BOTTOM_TRUE,
      BOTTOM_FALSE,
      BOTTOM_EQUAL_FALSE,
    ];

    for (const spec of SPECS) {
      it(spec.name, () => {
        const actual = needToGoAround(spec.props);
        expect(actual).toEqual(spec.expected);
      });
    }
  });

  describe("travelSegments", () => {
    interface Spec {
      name: string;
      source: xy.XY;
      segments: Segment[];
      expected: xy.XY;
    }

    const TO_RIGHT: Spec = {
      name: "to right",
      source: { x: 0, y: 0 },
      segments: [{ direction: "x", length: 10 }],
      expected: { x: 10, y: 0 },
    };

    const TO_LEFT: Spec = {
      name: "to left",
      source: { x: 10, y: 0 },
      segments: [{ direction: "x", length: -10 }],
      expected: { x: 0, y: 0 },
    };

    const TO_TOP: Spec = {
      name: "to top",
      source: { x: 0, y: 10 },
      segments: [{ direction: "y", length: -10 }],
      expected: { x: 0, y: 0 },
    };

    const TO_BOTTOM: Spec = {
      name: "to bottom",
      source: { x: 0, y: 0 },
      segments: [{ direction: "y", length: 10 }],
      expected: { x: 0, y: 10 },
    };

    const SPECS = [TO_RIGHT, TO_LEFT, TO_TOP, TO_BOTTOM];

    for (const spec of SPECS) {
      it(spec.name, () => {
        const actual = travelSegments(spec.source, ...spec.segments);
        expect(actual).toEqual(spec.expected);
      });
    }
  });

  describe("stump", () => {
    interface Spec {
      description: string;
      props: PrepareNodeProps;
      expected: Segment | undefined;
    }

    const LEFT_LEFT_TRUE: Spec = {
      description: `source has left orientation and is to right of target. target has right orientation
        and is to left of source`,
      props: {
        sourceStumpTip: { x: 0, y: 0 },
        targetStumpTip: { x: 10, y: 0 },
        sourceBox: box.ZERO,
        targetBox: box.construct({ x: 10, y: 0 }),
        sourceOrientation: "left",
        targetOrientation: "left",
      },
      expected: { direction: "y", length: -10 },
    };

    const LEFT_LEFT_FALSE: Spec = {
      description: "right to left",
      props: {
        sourceStumpTip: { x: 10, y: 0 },
        targetStumpTip: { x: 0, y: 0 },
        sourceBox: box.construct({ x: 10, y: 0 }),
        targetBox: box.ZERO,
        sourceOrientation: "left",
        targetOrientation: "left",
      },
      expected: undefined,
    };

    const RIGHT_LEFT_TRUE: Spec = {
      description: "right to left",
      props: {
        sourceStumpTip: { x: 10, y: 20 },
        targetStumpTip: { x: 0, y: 0 },
        sourceBox: box.construct({ x: 10, y: 0 }),
        targetBox: box.ZERO,
        sourceOrientation: "right",
        targetOrientation: "left",
      },
      expected: { direction: "y", length: -10 },
    };

    const RIGHT_LEFT_TRUE_LONG_WAY: Spec = {
      description: "right to left",
      props: {
        sourceStumpTip: { x: 10, y: 0 },
        targetStumpTip: { x: 0, y: 0 },
        sourceBox: box.construct({ x: 10, y: 0 }),
        targetBox: box.ZERO,
        sourceOrientation: "right",
        targetOrientation: "left",
      },
      expected: { direction: "y", length: 10 },
    };

    const SPECS = [
      LEFT_LEFT_TRUE,
      LEFT_LEFT_FALSE,
      RIGHT_LEFT_TRUE,
      RIGHT_LEFT_TRUE_LONG_WAY,
    ];

    for (const spec of SPECS) {
      it(spec.description, () => {
        const actual = prepareNode(spec.props);
        expect(actual).toEqual(spec.expected);
      });
    }
  });

  describe("new connector formation", () => {
    interface Spec {
      name: string;
      props: BuildNewConnectorProps;
      expected: Segment[];
    }
    const SIMPLE_BOTTOM_TO_TOP: Spec = {
      name: "simple bottom to top",
      props: {
        sourceOrientation: "bottom",
        targetOrientation: "top",
        sourcePos: { x: 0, y: 0 },
        targetPos: { x: 0, y: 30 },
        sourceBox: box.ZERO,
        targetBox: box.ZERO,
      },
      expected: [{ direction: "y", length: 30 }],
    };

    const SIMPLE_LEFT_TO_RIGHT: Spec = {
      name: "simple left to right",
      props: {
        sourceOrientation: "left",
        targetOrientation: "right",
        sourcePos: { x: 30, y: 0 },
        targetPos: { x: 0, y: 0 },
        sourceBox: box.ZERO,
        targetBox: box.ZERO,
      },
      expected: [{ direction: "x", length: -30 }],
    };

    const SIMPLE_TOP_TO_BOTTOM: Spec = {
      name: "simple top to bottom",
      props: {
        sourceOrientation: "top",
        targetOrientation: "bottom",
        sourcePos: { x: 0, y: 0 },
        targetPos: { x: 0, y: -30 },
        sourceBox: box.ZERO,
        targetBox: box.ZERO,
      },
      expected: [{ direction: "y", length: -30 }],
    };

    const SIMPLE_RIGHT_TO_LEFT: Spec = {
      name: "simple right to left",
      props: {
        sourceOrientation: "right",
        targetOrientation: "left",
        sourcePos: { x: 0, y: 0 },
        targetPos: { x: 30, y: 0 },
        sourceBox: box.ZERO,
        targetBox: box.ZERO,
      },
      expected: [{ direction: "x", length: 30 }],
    };

    const LEFT_LEFT_TARGET_DOWN_RIGHT: Spec = {
      name: "left and left - target is down and right",
      props: {
        sourceOrientation: "left",
        targetOrientation: "left",
        sourcePos: { x: 0, y: 0 },
        targetPos: { x: 30, y: 30 },
        sourceBox: box.ZERO,
        targetBox: box.ZERO,
      },
      expected: [
        // Left
        { direction: "x", length: -10 },
        // Down
        { direction: "y", length: 30 },
        // Right
        { direction: "x", length: 40 },
      ],
    };

    const LEFT_LEFT_TARGET_UP_LEFT: Spec = {
      name: "left and left - target is up and left",
      props: {
        sourceOrientation: "left",
        targetOrientation: "left",
        sourcePos: { x: 30, y: 30 },
        targetPos: { x: 0, y: 0 },
        sourceBox: box.ZERO,
        targetBox: box.ZERO,
      },
      expected: [
        // Left
        { direction: "x", length: -40 },
        // Up
        { direction: "y", length: -30 },
        // Right
        { direction: "x", length: 10 },
      ],
    };

    const LEFT_LEFT_TARGET_EQ_RIGHT: Spec = {
      name: "left and left - target is equal and right",
      props: {
        sourceOrientation: "left",
        targetOrientation: "left",
        sourcePos: { x: 0, y: 0 },
        targetPos: { x: 30, y: 0 },
        sourceBox: box.ZERO,
        targetBox: box.ZERO,
      },
      expected: [
        // Left
        { direction: "x", length: -10 },
        // Up
        { direction: "y", length: -10 },
        // Right
        { direction: "x", length: 30 },
        // Down
        { direction: "y", length: 10 },
        // Right
        { direction: "x", length: 10 },
      ],
    };

    const LEFT_LEFT_TARGET_EQ_LEFT: Spec = {
      name: "left and left - target is equal and left",
      props: {
        sourceOrientation: "left",
        targetOrientation: "left",
        sourcePos: { x: 30, y: 0 },
        targetPos: { x: 0, y: 0 },
        sourceBox: box.ZERO,
        targetBox: box.ZERO,
      },
      expected: [
        // Left
        { direction: "x", length: -10 },
        // Up
        { direction: "y", length: -10 },
        // Left
        { direction: "x", length: -30 },
        // Down
        { direction: "y", length: 10 },
        // Right
        { direction: "x", length: 10 },
      ],
    };

    const SPECS = [
      SIMPLE_BOTTOM_TO_TOP,
      SIMPLE_LEFT_TO_RIGHT,
      SIMPLE_TOP_TO_BOTTOM,
      SIMPLE_RIGHT_TO_LEFT,
      LEFT_LEFT_TARGET_DOWN_RIGHT,
      LEFT_LEFT_TARGET_UP_LEFT,
      LEFT_LEFT_TARGET_EQ_RIGHT,
      LEFT_LEFT_TARGET_EQ_LEFT,
    ];

    for (const spec of SPECS) {
      it(spec.name, () => {
        const actual = newConnector(spec.props);
        expect(actual).toEqual(spec.expected);
      });
    }
  });

  describe("move", () => {
    interface Spec {
      name: string;
      props: MoveConnectorProps;
      expected: Segment[];
    }

    const SINGLE_MOVE_UP: Spec = {
      name: "single move up",
      props: {
        segments: [{ direction: "x", length: 30 }],
        index: 0,
        magnitude: 10,
      },
      expected: [
        { direction: "x", length: 10 },
        { direction: "y", length: -10 },
        { direction: "x", length: 10 },
        { direction: "y", length: 10 },
        { direction: "x", length: 10 },
      ],
    };

    const SPECS: Spec[] = [SINGLE_MOVE_UP];

    for (const spec of SPECS) {
      it(spec.name, () => {
        const actual = moveConnector(spec.props);
        expect(actual).toEqual(spec.expected);
      });
    }
  });

  describe("removeShortSegments", () => {
    interface Spec {
      name: string;
      props:
    }
  })
});
