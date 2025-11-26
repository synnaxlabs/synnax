// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  alignNodes,
  distributeNodes,
  rotateNodes,
  rotateNodesAroundCenter,
} from "@/vis/diagram/align";
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
        new NodeLayout(
          "n1",
          box.construct({ x: 10, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 50, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 0, y: 100 }, { width: 100, height: 100 }),
          [],
        ),
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
        new NodeLayout(
          "n1",
          box.construct({ x: 20, y: 0 }, { width: 80, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 10, y: 50 }, { width: 120, height: 100 }),
          [],
        ),
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
        new NodeLayout(
          "n1",
          box.construct({ x: 10, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 50, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 0, y: 100 }, { width: 100, height: 100 }),
          [],
        ),
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
        new NodeLayout(
          "n1",
          box.construct({ x: 20, y: 0 }, { width: 80, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 10, y: 50 }, { width: 120, height: 100 }),
          [],
        ),
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
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 10 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 50, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 100, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
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
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 20 }, { width: 100, height: 80 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 50, y: 10 }, { width: 100, height: 120 }),
          [],
        ),
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
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 10 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 50, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 100, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
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
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 20 }, { width: 100, height: 80 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 50, y: 10 }, { width: 100, height: 120 }),
          [],
        ),
      ];
      const outputs = alignNodes(inputs, "bottom");
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 50 },
        { x: 50, y: 10 },
      ]);
    });
  });
});

describe("distribute", () => {
  describe("distribute horizontal", () => {
    it("should distribute three nodes with equal spacing", () => {
      const inputs = [
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 150, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 400, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
      ];
      const outputs = distributeNodes(inputs, "x");
      // Total space: 400 - 100 = 300
      // Middle width: 100
      // Gap size: (300 - 100) / 2 = 100
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 0 }, // First stays at 0
        { x: 200, y: 0 }, // Middle: 100 (first right) + 100 (gap) = 200
        { x: 400, y: 0 }, // Last stays at 400
      ]);
    });

    it("should distribute nodes with different widths", () => {
      const inputs = [
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 0 }, { width: 50, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 100, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 300, y: 0 }, { width: 50, height: 100 }),
          [],
        ),
      ];
      const outputs = distributeNodes(inputs, "x");
      // Total space: 300 - 50 = 250
      // Middle width: 100
      // Gap size: (250 - 100) / 2 = 75
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 0 }, // First stays
        { x: 125, y: 0 }, // 50 (first right) + 75 (gap) = 125
        { x: 300, y: 0 }, // Last stays
      ]);
    });

    it("should handle four nodes", () => {
      const inputs = [
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 150, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 250, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n4",
          box.construct({ x: 600, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
      ];
      const outputs = distributeNodes(inputs, "x");
      // Total space: 600 - 100 = 500
      // Middle widths: 100 + 100 = 200
      // Gap size: (500 - 200) / 3 = 100
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 0 }, // First stays
        { x: 200, y: 0 }, // 100 + 100 = 200
        { x: 400, y: 0 }, // 200 + 100 + 100 = 400
        { x: 600, y: 0 }, // Last stays
      ]);
    });

    it("should preserve y positions", () => {
      const inputs = [
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 150, y: 100 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 400, y: 25 }, { width: 100, height: 100 }),
          [],
        ),
      ];
      const outputs = distributeNodes(inputs, "x");
      expect(outputs.map((o) => box.top(o.box))).toEqual([50, 100, 25]);
    });

    it("should handle overlapping nodes with zero gap", () => {
      // Nodes that would overlap - should stack with 0 gap instead
      const inputs = [
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 50, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 100, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
      ];
      const outputs = distributeNodes(inputs, "x");
      // Total space: 100 - 100 = 0
      // Middle width: 100
      // Gap would be negative: (0 - 100) / 2 = -50
      // But Math.max(0, -50) = 0, so nodes touch with no overlap
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 0 }, // First stays at 0
        { x: 100, y: 0 }, // 100 (first right) + 0 (gap) = 100
        { x: 200, y: 0 }, // 100 + 100 (width) + 0 (gap) = 200
      ]);
    });

    it("should sort by vertical position when overlapping (except leftmost stays first)", () => {
      // Negative overlap: leftmost node stays first, rest sorted by vertical position
      const inputs = [
        new NodeLayout(
          "n1",
          box.construct({ x: 50, y: 100 }, { width: 100, height: 100 }),
          [],
        ), // Middle horizontally, middle vertically
        new NodeLayout(
          "n2",
          box.construct({ x: 0, y: 200 }, { width: 100, height: 100 }),
          [],
        ), // Leftmost, bottom
        new NodeLayout(
          "n3",
          box.construct({ x: 100, y: 0 }, { width: 100, height: 100 }),
          [],
        ), // Rightmost, top
      ];
      const outputs = distributeNodes(inputs, "x");
      // n2 is leftmost (x=0), so it's positioned first at x=0
      // Remaining nodes sorted by y: n3 (y=0) at x=100, n1 (y=100) at x=200
      // Check each node's final position by finding it by key
      const n1 = outputs.find((o) => o.key === "n1")!;
      const n2 = outputs.find((o) => o.key === "n2")!;
      const n3 = outputs.find((o) => o.key === "n3")!;

      expect(box.topLeft(n2.box)).toEqual({ x: 0, y: 200 }); // n2: leftmost, at x=0
      expect(box.topLeft(n3.box)).toEqual({ x: 100, y: 0 }); // n3: sorted second (top), at x=100
      expect(box.topLeft(n1.box)).toEqual({ x: 200, y: 100 }); // n1: sorted third (middle), at x=200
    });
  });

  describe("distribute vertical", () => {
    it("should distribute three nodes with equal spacing", () => {
      const inputs = [
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 0, y: 150 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 0, y: 400 }, { width: 100, height: 100 }),
          [],
        ),
      ];
      const outputs = distributeNodes(inputs, "y");
      // Total space: 400 - 100 = 300
      // Middle height: 100
      // Gap size: (300 - 100) / 2 = 100
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 0 }, // First stays at 0
        { x: 0, y: 200 }, // Middle: 100 (first bottom) + 100 (gap) = 200
        { x: 0, y: 400 }, // Last stays at 400
      ]);
    });

    it("should distribute nodes with different heights", () => {
      const inputs = [
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 50 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 0, y: 100 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 0, y: 300 }, { width: 100, height: 50 }),
          [],
        ),
      ];
      const outputs = distributeNodes(inputs, "y");
      // Total space: 300 - 50 = 250
      // Middle height: 100
      // Gap size: (250 - 100) / 2 = 75
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 0 }, // First stays
        { x: 0, y: 125 }, // 50 (first bottom) + 75 (gap) = 125
        { x: 0, y: 300 }, // Last stays
      ]);
    });

    it("should preserve x positions", () => {
      const inputs = [
        new NodeLayout(
          "n1",
          box.construct({ x: 50, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 100, y: 150 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 25, y: 400 }, { width: 100, height: 100 }),
          [],
        ),
      ];
      const outputs = distributeNodes(inputs, "y");
      expect(outputs.map((o) => box.left(o.box))).toEqual([50, 100, 25]);
    });

    it("should handle overlapping nodes with zero gap", () => {
      // Nodes that would overlap - should stack with 0 gap instead
      const inputs = [
        new NodeLayout(
          "n1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n2",
          box.construct({ x: 0, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "n3",
          box.construct({ x: 0, y: 100 }, { width: 100, height: 100 }),
          [],
        ),
      ];
      const outputs = distributeNodes(inputs, "y");
      // Total space: 100 - 100 = 0
      // Middle height: 100
      // Gap would be negative: (0 - 100) / 2 = -50
      // But Math.max(0, -50) = 0, so nodes touch with no overlap
      expect(outputs.map((o) => box.topLeft(o.box))).toEqual([
        { x: 0, y: 0 }, // First stays at 0
        { x: 0, y: 100 }, // 100 (first bottom) + 0 (gap) = 100
        { x: 0, y: 200 }, // 100 + 100 (height) + 0 (gap) = 200
      ]);
    });
  });
});

describe("rotate", () => {
  it("should return layouts unchanged (rotation handled at console level)", () => {
    const inputs = [
      new NodeLayout("n1", box.construct(xy.ZERO, { width: 100, height: 100 }), []),
      new NodeLayout(
        "n2",
        box.construct({ x: 150, y: 0 }, { width: 100, height: 100 }),
        [],
      ),
    ];

    const result = rotateNodes(inputs, "clockwise");
    expect(result).toBe(inputs);
  });
});

describe("rotateNodesAroundCenter", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    vi.clearAllMocks();
  });

  it("should return empty array for empty input", () => {
    const result = rotateNodesAroundCenter([], "clockwise");
    expect(result).toEqual([]);
  });

  it("should rotate a single node around itself (stays in place)", () => {
    const inputs = [
      new NodeLayout(
        "n1",
        box.construct({ x: 100, y: 100 }, { width: 50, height: 50 }),
        [],
      ),
    ];
    const outputs = rotateNodesAroundCenter(inputs, "clockwise");

    // Single node's center is the group center, so it stays at same position
    expect(box.topLeft(outputs[0].box)).toEqual({ x: 100, y: 100 });
  });

  it("should rotate two nodes 90 degrees clockwise around their center", () => {
    const inputs = [
      new NodeLayout(
        "n1",
        box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
        [],
      ),
      new NodeLayout(
        "n2",
        box.construct({ x: 200, y: 0 }, { width: 100, height: 100 }),
        [],
      ),
    ];
    const outputs = rotateNodesAroundCenter(inputs, "clockwise");

    // Group center: x=(0+300)/2=150, y=(0+100)/2=50
    // n1 center (50,50) relative to group center: (-100, 0)
    // After 90° CW: (0, -100) absolute: (150, -50), top-left: (100, -100)
    expect(box.topLeft(outputs[0].box)).toEqual({ x: 100, y: -100 });

    // n2 center (250,50) relative to group center: (100, 0)
    // After 90° CW: (0, 100) absolute: (150, 150), top-left: (100, 100)
    expect(box.topLeft(outputs[1].box)).toEqual({ x: 100, y: 100 });
  });

  it("should rotate two nodes 90 degrees counter-clockwise around their center", () => {
    const inputs = [
      new NodeLayout(
        "n1",
        box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
        [],
      ),
      new NodeLayout(
        "n2",
        box.construct({ x: 200, y: 0 }, { width: 100, height: 100 }),
        [],
      ),
    ];
    const outputs = rotateNodesAroundCenter(inputs, "counterclockwise");

    // Group center: x=150, y=50
    // n1 center (50,50) relative to group center: (-100, 0)
    // After 90° CCW: (0, 100) absolute: (150, 150), top-left: (100, 100)
    expect(box.topLeft(outputs[0].box)).toEqual({ x: 100, y: 100 });

    // n2 center (250,50) relative to group center: (100, 0)
    // After 90° CCW: (0, -100) absolute: (150, -50), top-left: (100, -100)
    expect(box.topLeft(outputs[1].box)).toEqual({ x: 100, y: -100 });
  });

  it("should preserve node dimensions after rotation", () => {
    const inputs = [
      new NodeLayout(
        "n1",
        box.construct({ x: 0, y: 0 }, { width: 150, height: 80 }),
        [],
      ),
      new NodeLayout(
        "n2",
        box.construct({ x: 200, y: 0 }, { width: 100, height: 120 }),
        [],
      ),
    ];
    const outputs = rotateNodesAroundCenter(inputs, "clockwise");

    expect(box.dims(outputs[0].box)).toEqual({ width: 150, height: 80 });
    expect(box.dims(outputs[1].box)).toEqual({ width: 100, height: 120 });
  });

  it("should handle three nodes in L-shape", () => {
    const inputs = [
      new NodeLayout(
        "n1",
        box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
        [],
      ),
      new NodeLayout(
        "n2",
        box.construct({ x: 100, y: 0 }, { width: 100, height: 100 }),
        [],
      ),
      new NodeLayout(
        "n3",
        box.construct({ x: 0, y: 100 }, { width: 100, height: 100 }),
        [],
      ),
    ];
    const outputs = rotateNodesAroundCenter(inputs, "clockwise");

    // Verify all nodes have moved to new positions
    expect(outputs).toHaveLength(3);
    const positions = outputs.map((r) => box.topLeft(r.box));

    // After rotation, positions should be different from original
    expect(positions[0]).not.toEqual({ x: 0, y: 0 });
    expect(positions[1]).not.toEqual({ x: 100, y: 0 });
    expect(positions[2]).not.toEqual({ x: 0, y: 100 });
  });
});
