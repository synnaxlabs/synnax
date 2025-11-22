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

import { alignNodes } from "@/vis/diagram/align";
import { HandleLayout, NodeLayout } from "@/vis/diagram/util";

describe("align", () => {
  interface Spec {
    name: string;
    inputs: NodeLayout[];
    outputs: xy.XY[];
  }

  const TWO_NODES_SAME_DIMS_SAME_HANDLES: Spec = {
    name: "nodes have same dimensions and handle positions",
    inputs: [
      new NodeLayout("n1", box.construct(xy.ZERO, { width: 100, height: 100 }), [
        new HandleLayout({ x: 0, y: 50 }, "left"),
        new HandleLayout({ x: 100, y: 50 }, "right"),
      ]),
      new NodeLayout(
        "n2",
        box.construct({ x: 10, y: 10 }, { width: 100, height: 100 }),
        [
          new HandleLayout({ x: 0, y: 50 }, "left"),
          new HandleLayout({ x: 100, y: 50 }, "right"),
        ],
      ),
    ],
    outputs: [
      { x: 0, y: 0 },
      { x: 10, y: 0 },
    ],
  };

  const TWO_NODES_SAME_DIMS_DIFF_HANDLES: Spec = {
    name: "nodes have same dimensions but different handle positions",
    inputs: [
      new NodeLayout("n1", box.construct(xy.ZERO, { width: 100, height: 100 }), [
        new HandleLayout({ x: 0, y: 50 }, "left"),
        new HandleLayout({ x: 100, y: 50 }, "right"),
      ]),
      new NodeLayout(
        "n2",
        box.construct({ x: 10, y: 10 }, { width: 100, height: 100 }),
        [
          new HandleLayout({ x: 0, y: 60 }, "left"),
          new HandleLayout({ x: 100, y: 50 }, "right"),
        ],
      ),
    ],
    outputs: [
      { x: 0, y: 0 },
      { x: 10, y: -10 },
    ],
  };

  const TWO_NODES_DIFF_DIMS_SAME_HANDLES: Spec = {
    name: "nodes have different dimensions but same handle positions",
    inputs: [
      new NodeLayout("n1", box.construct(xy.ZERO, { width: 100, height: 100 }), [
        new HandleLayout({ x: 0, y: 50 }, "left"),
        new HandleLayout({ x: 100, y: 50 }, "right"),
      ]),
      new NodeLayout(
        "n2",
        box.construct({ x: 10, y: 10 }, { width: 120, height: 120 }),
        [
          new HandleLayout({ x: 0, y: 50 }, "left"),
          new HandleLayout({ x: 100, y: 50 }, "right"),
        ],
      ),
    ],
    outputs: [
      { x: 0, y: 0 },
      { x: 10, y: 0 },
    ],
  };

  const THREE_NODES_SAME_DIMS_SAME_HANDLES: Spec = {
    name: "nodes have same dimensions and handle positions",
    inputs: [
      new NodeLayout("n1", box.construct(xy.ZERO, { width: 100, height: 100 }), [
        new HandleLayout({ x: 0, y: 50 }, "left"),
        new HandleLayout({ x: 100, y: 50 }, "right"),
      ]),
      new NodeLayout(
        "n2",
        box.construct({ x: 10, y: 10 }, { width: 100, height: 100 }),
        [
          new HandleLayout({ x: 0, y: 50 }, "left"),
          new HandleLayout({ x: 100, y: 50 }, "right"),
        ],
      ),
      new NodeLayout(
        "n3",
        box.construct({ x: 20, y: 20 }, { width: 100, height: 100 }),
        [
          new HandleLayout({ x: 0, y: 50 }, "left"),
          new HandleLayout({ x: 100, y: 50 }, "right"),
        ],
      ),
    ],
    outputs: [
      { x: 0, y: 0 },
      { x: 10, y: 0 },
      { x: 20, y: 0 },
    ],
  };

  const THREE_NODES_SAME_DIMS_DIFF_HANDLES: Spec = {
    name: "nodes have same dimensions but different handle positions",
    inputs: [
      new NodeLayout("n1", box.construct(xy.ZERO, { width: 100, height: 100 }), [
        new HandleLayout({ x: 0, y: 50 }, "left"),
        new HandleLayout({ x: 100, y: 50 }, "right"),
      ]),
      new NodeLayout(
        "n2",
        box.construct({ x: 10, y: 10 }, { width: 100, height: 100 }),
        [
          new HandleLayout({ x: 0, y: 60 }, "left"),
          new HandleLayout({ x: 100, y: 50 }, "right"),
        ],
      ),
      new NodeLayout(
        "n3",
        box.construct({ x: 20, y: 20 }, { width: 100, height: 100 }),
        [
          new HandleLayout({ x: 0, y: 40 }, "left"),
          new HandleLayout({ x: 100, y: 50 }, "right"),
        ],
      ),
    ],
    outputs: [
      { x: 0, y: 0 },
      { x: 10, y: -10 },
      { x: 20, y: 0 },
    ],
  };

  const SPECS = [
    TWO_NODES_SAME_DIMS_SAME_HANDLES,
    TWO_NODES_SAME_DIMS_DIFF_HANDLES,
    TWO_NODES_DIFF_DIMS_SAME_HANDLES,
    THREE_NODES_SAME_DIMS_SAME_HANDLES,
    THREE_NODES_SAME_DIMS_DIFF_HANDLES,
  ];

  for (const spec of SPECS)
    it(spec.name, () => {
      const outputs = alignNodes(spec.inputs);
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual(spec.outputs);
    });

  describe("align left", () => {
    it("should align all nodes to the leftmost node's left edge", () => {
      const inputs = [
        new NodeLayout("n1", box.construct({ x: 10, y: 0 }, { width: 100, height: 100 }), []),
        new NodeLayout("n2", box.construct({ x: 50, y: 50 }, { width: 100, height: 100 }), []),
        new NodeLayout("n3", box.construct({ x: 0, y: 100 }, { width: 100, height: 100 }), []),
      ];
      const outputs = alignNodes(inputs, "left");
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 0 },
        { x: 0, y: 50 },
        { x: 0, y: 100 },
      ]);
    });

    it("should handle nodes with different widths", () => {
      const inputs = [
        new NodeLayout("n1", box.construct({ x: 20, y: 0 }, { width: 80, height: 100 }), []),
        new NodeLayout("n2", box.construct({ x: 10, y: 50 }, { width: 120, height: 100 }), []),
      ];
      const outputs = alignNodes(inputs, "left");
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 10, y: 0 },
        { x: 10, y: 50 },
      ]);
    });
  });

  describe("align right", () => {
    it("should align all nodes to the rightmost node's right edge", () => {
      const inputs = [
        new NodeLayout("n1", box.construct({ x: 10, y: 0 }, { width: 100, height: 100 }), []),
        new NodeLayout("n2", box.construct({ x: 50, y: 50 }, { width: 100, height: 100 }), []),
        new NodeLayout("n3", box.construct({ x: 0, y: 100 }, { width: 100, height: 100 }), []),
      ];
      const outputs = alignNodes(inputs, "right");
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 50, y: 0 },
        { x: 50, y: 50 },
        { x: 50, y: 100 },
      ]);
    });

    it("should handle nodes with different widths", () => {
      const inputs = [
        new NodeLayout("n1", box.construct({ x: 20, y: 0 }, { width: 80, height: 100 }), []),
        new NodeLayout("n2", box.construct({ x: 10, y: 50 }, { width: 120, height: 100 }), []),
      ];
      const outputs = alignNodes(inputs, "right");
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 50, y: 0 },
        { x: 10, y: 50 },
      ]);
    });
  });

  describe("align top", () => {
    it("should align all nodes to the topmost node's top edge", () => {
      const inputs = [
        new NodeLayout("n1", box.construct({ x: 0, y: 10 }, { width: 100, height: 100 }), []),
        new NodeLayout("n2", box.construct({ x: 50, y: 50 }, { width: 100, height: 100 }), []),
        new NodeLayout("n3", box.construct({ x: 100, y: 0 }, { width: 100, height: 100 }), []),
      ];
      const outputs = alignNodes(inputs, "top");
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 0 },
        { x: 50, y: 0 },
        { x: 100, y: 0 },
      ]);
    });

    it("should handle nodes with different heights", () => {
      const inputs = [
        new NodeLayout("n1", box.construct({ x: 0, y: 20 }, { width: 100, height: 80 }), []),
        new NodeLayout("n2", box.construct({ x: 50, y: 10 }, { width: 100, height: 120 }), []),
      ];
      const outputs = alignNodes(inputs, "top");
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 10 },
        { x: 50, y: 10 },
      ]);
    });
  });

  describe("align bottom", () => {
    it("should align all nodes to the bottommost node's bottom edge", () => {
      const inputs = [
        new NodeLayout("n1", box.construct({ x: 0, y: 10 }, { width: 100, height: 100 }), []),
        new NodeLayout("n2", box.construct({ x: 50, y: 50 }, { width: 100, height: 100 }), []),
        new NodeLayout("n3", box.construct({ x: 100, y: 0 }, { width: 100, height: 100 }), []),
      ];
      const outputs = alignNodes(inputs, "bottom");
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 50 },
        { x: 50, y: 50 },
        { x: 100, y: 50 },
      ]);
    });

    it("should handle nodes with different heights", () => {
      const inputs = [
        new NodeLayout("n1", box.construct({ x: 0, y: 20 }, { width: 100, height: 80 }), []),
        new NodeLayout("n2", box.construct({ x: 50, y: 10 }, { width: 100, height: 120 }), []),
      ];
      const outputs = alignNodes(inputs, "bottom");
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 50 },
        { x: 50, y: 10 },
      ]);
    });
  });
});
