// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { connector } from "@/schematic/edge/connector/index";

describe("connector", () => {
  describe("needToGoAroundSource", () => {
    interface spec {
      name: string;
      props: connector.NeedToGoAroundSourceProps;
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

    for (const spec of SPECS)
      it(spec.name, () => {
        const actual = connector.needToGoAround(spec.props);
        expect(actual).toEqual(spec.expected);
      });
  });

  describe("travelSegments", () => {
    interface Spec {
      name: string;
      source: xy.XY;
      segments: connector.Segment[];
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

    for (const spec of SPECS)
      it(spec.name, () => {
        const actual = connector.travelSegments(spec.source, ...spec.segments);
        expect(actual).toEqual(spec.expected);
      });
  });

  describe("stump", () => {
    interface Spec {
      description: string;
      props: connector.PrepareNodeProps;
      expected: connector.Segment | undefined;
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
      expected: { direction: "y", length: -connector.STUMP_LENGTH },
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
      expected: { direction: "y", length: -connector.STUMP_LENGTH },
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

    for (const spec of SPECS)
      it(spec.description, () => {
        const actual = connector.prepareNode(spec.props);
        expect(actual).toEqual(spec.expected);
      });
  });

  describe("new connector formation", () => {
    interface Spec {
      name: string;
      props: connector.BuildNew;
      expected: connector.Segment[];
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

    for (const spec of SPECS)
      it(spec.name, () => {
        const actual = connector.buildNew(spec.props);
        expect(actual).toEqual(spec.expected);
        // We also want to do a sanity check to make sure that the connector actually gets to the target from the
        // source.
        const target = connector.travelSegments(spec.props.sourcePos, ...actual);
        expect(target).toEqual(spec.props.targetPos);
      });
  });

  describe("dragging segments", () => {
    interface Spec {
      name: string;
      props: connector.MoveConnectorProps;
      expected: connector.Segment[];
    }

    // Props:
    // S---T
    //
    // Expected:
    //
    // S-| |-T
    //   |-|
    //
    const SINGLE_MOVE_UP: Spec = {
      name: "single move up",
      props: { segments: [{ direction: "x", length: 30 }], index: 0, magnitude: 10 },
      expected: [
        { direction: "x", length: 10 },
        { direction: "y", length: 10 },
        { direction: "x", length: 10 },
        { direction: "y", length: -10 },
        { direction: "x", length: 10 },
      ],
    };

    // Props:
    //      T
    // S-|  |
    //   |  |
    //   |--|
    //
    // Expected:
    //      T
    // S-|  |
    //   |--|
    const U_SHAPED_DRAG_UP_NO_COMPRESSION: Spec = {
      name: "u shaped drag up no compression",
      props: {
        segments: [
          { direction: "x", length: 10 },
          { direction: "y", length: 20 },
          { direction: "x", length: 10 },
          { direction: "y", length: -20 },
        ],
        index: 2,
        magnitude: -10,
      },
      expected: [
        { direction: "x", length: 10 },
        { direction: "y", length: 10 },
        { direction: "x", length: 10 },
        { direction: "y", length: -10 },
      ],
    };

    // Props:
    //      T
    // S-|  |
    //   |--|
    //
    // Expected:
    //      T
    // S----|
    const U_SHAPED_DRAG_UP_COMPRESSION: Spec = {
      name: "u shaped drag up compression",
      props: {
        segments: [
          { direction: "x", length: 10 },
          { direction: "y", length: 10 },
          { direction: "x", length: 10 },
          { direction: "y", length: -12 },
        ],
        index: 2,
        magnitude: -10,
      },
      expected: [
        { direction: "x", length: 20 },
        { direction: "y", length: -2 },
      ],
    };

    // Props:
    //  |--S
    //  |-----T
    //
    // Expected:
    //     |-S
    //   |-|
    //   |-----T
    //
    const ADD_SOURCE_STUMP: Spec = {
      name: "add source stump",
      props: {
        segments: [
          { direction: "x", length: -20 },
          { direction: "y", length: 20 },
          { direction: "x", length: 30 },
        ],
        index: 0,
        magnitude: 10,
      },
      expected: [
        { direction: "x", length: -10 },
        { direction: "y", length: 10 },
        { direction: "x", length: -10 },
        { direction: "y", length: 10 },
        { direction: "x", length: 30 },
      ],
    };

    // Props:
    // |--S
    // |-T
    // Expected:
    // |--S
    // |T
    const SPECS: Spec[] = [
      SINGLE_MOVE_UP,
      U_SHAPED_DRAG_UP_NO_COMPRESSION,
      U_SHAPED_DRAG_UP_COMPRESSION,
      ADD_SOURCE_STUMP,
    ];

    for (const spec of SPECS)
      it(spec.name, () => {
        const actual = connector.dragSegment(spec.props);
        const propsTarget = connector.travelSegments(xy.ZERO, ...spec.props.segments);
        const expectedTarget = connector.travelSegments(xy.ZERO, ...spec.expected);
        const actualTarget = connector.travelSegments(xy.ZERO, ...actual);
        expect(propsTarget).toEqual(expectedTarget);
        expect(actual).toEqual(spec.expected);
        expect(propsTarget).toEqual(actualTarget);
      });
  });

  describe("moving nodes", () => {
    interface Spec {
      name: string;
      props: connector.MoveNodeProps;
      expected: connector.Segment[];
    }

    // Props:
    // S---T
    //
    // Expected:
    //
    // S-|
    //   |-T
    const SINGLE_MOVE_UP: Spec = {
      name: "single move up",
      props: { delta: { x: 0, y: -10 }, segments: [{ direction: "x", length: 30 }] },
      expected: [
        { direction: "x", length: 15 },
        { direction: "y", length: 10 },
        { direction: "x", length: 15 },
      ],
    };

    // Props:
    // S-|
    //   |-T
    //
    // Expected:
    // S---T
    const SINGLE_COMPRESS_DOWN: Spec = {
      name: "single compress down",
      props: {
        delta: { x: 0, y: 10 },
        segments: [
          { direction: "x", length: 15 },
          { direction: "y", length: 10 },
          { direction: "x", length: 15 },
        ],
      },
      expected: [{ direction: "x", length: 30 }],
    };

    // Props:
    // S---T
    //
    // Expected:
    // S-----T
    const SIMPLE_MOVE_LEFT: Spec = {
      name: "simple move left",
      props: { delta: { x: -10, y: 0 }, segments: [{ direction: "x", length: 30 }] },
      expected: [{ direction: "x", length: 40 }],
    };

    // Props:
    // S
    // |
    // T
    //
    // Expected:
    // S
    // |
    // |
    // T
    const SIMPLE_MOVE_UP: Spec = {
      name: "simple move up",
      props: { delta: { x: 0, y: -10 }, segments: [{ direction: "y", length: 30 }] },
      expected: [{ direction: "y", length: 40 }],
    };

    // Props:
    // |--S
    // |-----T
    //
    // Expected:
    // -S---T
    const OPPOSITE_ORIENTATION_COMPRESSION: Spec = {
      name: "opposite orientation compression",
      props: {
        delta: { x: 0, y: 20 },
        segments: [
          { direction: "x", length: -20 },
          { direction: "y", length: 20 },
          { direction: "x", length: 30 },
        ],
      },
      expected: [
        { direction: "x", length: -10 },
        { direction: "x", length: 20 },
      ],
    };

    // Props:
    // -S---T
    //
    // Expected:
    // |-----T
    // |-S
    const OPPOSITE_ORIENTATION_COMPRESSED_DISCONNECT: Spec = {
      name: "opposite orientation compressed disconnect",
      props: {
        delta: { x: 0, y: 20 },
        segments: [
          { direction: "x", length: -20 },
          { direction: "x", length: 30 },
        ],
      },
      expected: [
        { direction: "x", length: -20 },
        { direction: "y", length: -20 },
        { direction: "x", length: 30 },
      ],
    };

    // Props:
    // |-S
    // |---T-
    //
    // Expected:
    // -S--T-
    const DOUBLE_OPPOSITE_ORIENTATION_COMPRESSION: Spec = {
      name: "double opposite orientation compression",
      props: {
        delta: { x: 0, y: 20 },
        segments: [
          { direction: "x", length: -10 },
          { direction: "y", length: 20 },
          { direction: "x", length: 30 },
          { direction: "x", length: -10 },
        ],
      },
      expected: [
        { direction: "x", length: -10 },
        { direction: "x", length: 30 },
        { direction: "x", length: -10 },
      ],
    };

    // Props:
    // -S--T-
    //
    // Expected:
    // |-S
    // |---T-
    const DOUBLE_OPPOSITE_ORIENTATION_COMPRESSION_DISCONNECT: Spec = {
      name: "double opposite orientation compression disconnect",
      props: {
        delta: { x: 0, y: -20 },
        segments: [
          { direction: "x", length: -10 },
          { direction: "x", length: 30 },
          { direction: "x", length: -10 },
        ],
      },
      expected: [
        { direction: "x", length: -10 },
        { direction: "y", length: 20 },
        { direction: "x", length: 30 },
        { direction: "x", length: -10 },
      ],
    };

    const PARALLEL_ORIENTATION_COMPRESSION: Spec = {
      name: "parallel orientation compression",
      props: {
        delta: { x: 1, y: 0 },
        segments: [
          { direction: "x", length: -10 },
          { direction: "y", length: 20 },
          { direction: "x", length: -10 },
        ],
      },
      expected: [
        { direction: "x", length: -11 },
        { direction: "y", length: 20 },
        { direction: "x", length: -10 },
      ],
    };

    const TIGHT_SINGLE_COMPRESSION: Spec = {
      name: "tight single compression",
      props: { delta: { x: 1, y: 0 }, segments: [{ direction: "x", length: 6 }] },
      expected: [{ direction: "x", length: 5 }],
    };

    const ORTHOGONAL_DOWN: Spec = {
      name: "orthogonal down",
      props: {
        delta: { x: 0, y: 4 },
        segments: [
          { direction: "x", length: 30 },
          { direction: "y", length: 10 },
        ],
      },
      expected: [
        { direction: "x", length: 30 },
        { direction: "y", length: 6 },
      ],
    };

    const TIGHT_COMPRESSION: Spec = {
      name: "orthogonal reverse stump -2 ",
      props: {
        delta: { x: 5, y: 0 },
        segments: [
          { direction: "x", length: 2 },
          { direction: "y", length: -133 },
          { direction: "x", length: 9 },
        ],
      },
      expected: [
        { direction: "x", length: 2 },
        { direction: "y", length: -133 },
        { direction: "x", length: 4 },
      ],
    };

    const SPECS: Spec[] = [
      SINGLE_MOVE_UP,
      SINGLE_COMPRESS_DOWN,
      SIMPLE_MOVE_LEFT,
      SIMPLE_MOVE_UP,
      OPPOSITE_ORIENTATION_COMPRESSION,
      DOUBLE_OPPOSITE_ORIENTATION_COMPRESSION,
      DOUBLE_OPPOSITE_ORIENTATION_COMPRESSION_DISCONNECT,
      OPPOSITE_ORIENTATION_COMPRESSED_DISCONNECT,
      PARALLEL_ORIENTATION_COMPRESSION,
      TIGHT_SINGLE_COMPRESSION,
      ORTHOGONAL_DOWN,
      TIGHT_COMPRESSION,
    ];
    for (const spec of SPECS)
      it(spec.name, () => {
        const actual = connector.moveSourceNode(spec.props);
        const expectedTarget = connector.travelSegments(xy.ZERO, ...spec.expected);
        const actualTarget = connector.travelSegments(xy.ZERO, ...actual);
        expect(actual).toEqual(spec.expected);
        expect(actualTarget).toEqual(expectedTarget);
      });
  });
});
