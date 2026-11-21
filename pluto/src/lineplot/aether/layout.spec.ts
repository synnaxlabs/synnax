// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { collapse, join, Layout, spans, type Trace } from "@/lineplot/aether/layout";

describe("Layout", () => {
  describe("one segment", () => {
    const layout = new Layout([{ lower: 100, upper: 200 }], box.DECIMAL);

    it("should map the segment bounds onto the home view", () => {
      expect(layout.home(100)).toBeCloseTo(0);
      expect(layout.home(150)).toBeCloseTo(0.5);
      expect(layout.home(200)).toBeCloseTo(1);
    });

    it("should extend past the segment bounds", () => {
      expect(layout.home(50)).toBeCloseTo(-0.5);
      expect(layout.home(300)).toBeCloseTo(2);
      expect(layout.reverseHome(-0.5)).toBeCloseTo(50);
      expect(layout.reverseHome(2)).toBeCloseTo(300);
    });

    it("should give the bounds of the segment", () => {
      expect(layout.bounds).toEqual({ lower: 100, upper: 200 });
    });

    it("should map positions across the viewport", () => {
      const zoomed = new Layout(
        [{ lower: 100, upper: 200 }],
        box.construct({ x: 0.5, y: 0 }, { width: 0.25, height: 1 }),
      );
      expect(zoomed.pos(150)).toBeCloseTo(0);
      expect(zoomed.pos(175)).toBeCloseTo(1);
      expect(zoomed.reverse(0.5)).toBeCloseTo(162.5);
      expect(zoomed.segments[0].scale.pos(175)).toBeCloseTo(1);
    });
  });

  describe("several segments", () => {
    // Spans of 10, 30, and 60 share the home view as 0.1, 0.3, and 0.6.
    const layout = new Layout(
      [
        { lower: 0, upper: 10 },
        { lower: 1000, upper: 1030 },
        { lower: 5000, upper: 5060 },
      ],
      box.DECIMAL,
    );

    it("should place each segment after the one before it", () => {
      expect(layout.home(0)).toBeCloseTo(0);
      expect(layout.home(10)).toBeCloseTo(0.1);
      expect(layout.home(1000)).toBeCloseTo(0.1);
      expect(layout.home(1015)).toBeCloseTo(0.25);
      expect(layout.home(5000)).toBeCloseTo(0.4);
      expect(layout.home(5060)).toBeCloseTo(1);
    });

    it("should map a value between two segments to where they meet", () => {
      expect(layout.home(500)).toBeCloseTo(0.1);
      expect(layout.home(4000)).toBeCloseTo(0.4);
    });

    it("should map a position where two segments meet to the later one", () => {
      expect(layout.reverseHome(0.1)).toBeCloseTo(1000);
      expect(layout.reverseHome(0.4)).toBeCloseTo(5000);
    });

    it("should map a span that ends where two segments meet to the earlier one", () => {
      const s = layout.reverseHomeSpan({
        lower: 0.05,
        upper: layout.segments[1].position.lower,
      });
      expect(s.lower).toBeCloseTo(5);
      expect(s.upper).toBeCloseTo(10);
    });

    it("should map a span that starts where two segments meet to the later one", () => {
      const s = layout.reverseHomeSpan({
        lower: layout.segments[1].position.lower,
        upper: 0.25,
      });
      expect(s.lower).toBeCloseTo(1000);
      expect(s.upper).toBeCloseTo(1015);
    });

    it("should map positions inside each segment back to their values", () => {
      [5, 1020, 5030].forEach((v) =>
        expect(layout.reverseHome(layout.home(v))).toBeCloseTo(v),
      );
    });

    it("should give each segment one scale per unit of value", () => {
      layout.segments.forEach(({ scale }) => expect(scale.dim(10)).toBeCloseTo(0.1));
    });

    it("should give the bounds from the first segment to the last", () => {
      expect(layout.bounds).toEqual({ lower: 0, upper: 5060 });
    });
  });
});

const MINUTE = Number(TimeSpan.MINUTE.valueOf());
const SECOND = Number(TimeSpan.SECOND.valueOf());

const trace = (
  key: string,
  lower: number,
  upper: number,
  extra: Partial<Trace> = {},
): Trace => ({ key, group: "a", bounds: { lower, upper }, loading: false, ...extra });

describe("spans", () => {
  describe("time", () => {
    it("should split the time between two windows at its middle", () => {
      const s = spans([trace("1", 0, 10), trace("2", 30, 40)], "time");
      expect(s.get("1")).toEqual({ lower: -Infinity, upper: 20 });
      expect(s.get("2")).toEqual({ lower: 20, upper: Infinity });
    });

    it("should give an overlap to the window that starts first", () => {
      const s = spans([trace("2", 5, 20), trace("1", 0, 10)], "time");
      expect(s.get("1")).toEqual({ lower: -Infinity, upper: 10 });
      expect(s.get("2")).toEqual({ lower: 10, upper: Infinity });
    });

    it("should give no time to a window that an earlier window holds", () => {
      const s = spans([trace("1", 0, 100), trace("2", 10, 20)], "time");
      expect(s.get("1")).toEqual(bounds.INFINITE);
      const empty = s.get("2");
      expect(empty?.lower).toEqual(empty?.upper);
    });

    it("should give a rolling window only the time after the fixed windows", () => {
      const s = spans(
        [trace("live", 0, 100, { edge: "rolling" }), trace("fixed", 40, 60)],
        "time",
      );
      expect(s.get("fixed")).toEqual({ lower: -Infinity, upper: 60 });
      expect(s.get("live")).toEqual({ lower: 60, upper: Infinity });
    });

    it("should split each group on its own", () => {
      const s = spans(
        [trace("1", 0, 10), trace("2", 0, 10, { group: "b" }), trace("3", 30, 40)],
        "time",
      );
      expect(s.get("1")).toEqual({ lower: -Infinity, upper: 20 });
      expect(s.get("2")).toEqual(bounds.INFINITE);
    });

    it("should give a trace with no group the time of its own group", () => {
      const s = spans(
        [trace("1", 0, 10, { group: undefined }), trace("2", 30, 40)],
        "time",
      );
      expect(s.get("1")).toEqual(bounds.INFINITE);
    });

    it("should give no span to a trace with no window", () => {
      expect(spans([trace("1", Infinity, -Infinity)], "time").size).toBe(0);
    });

    it("should keep each span inside the segment of its window", () => {
      const segs = [
        { lower: -1, upper: 11 },
        { lower: 90, upper: 110 },
      ];
      const s = spans([trace("1", 0, 10), trace("2", 95, 105)], "time", segs);
      expect(s.get("1")).toEqual({ lower: -Infinity, upper: 11 });
      expect(s.get("2")).toEqual({ lower: 90, upper: Infinity });
    });

    it("should split the time inside one segment at its middle", () => {
      const segs = [
        { lower: -1, upper: 41 },
        { lower: 90, upper: 110 },
      ];
      const s = spans(
        [trace("1", 0, 10), trace("2", 30, 40), trace("3", 95, 105)],
        "time",
        segs,
      );
      expect(s.get("1")).toEqual({ lower: -Infinity, upper: 20 });
      expect(s.get("2")).toEqual({ lower: 20, upper: 41 });
      expect(s.get("3")).toEqual({ lower: 90, upper: Infinity });
    });
  });

  describe("relative", () => {
    it("should pad each window by a share of its span on each side", () => {
      const s = spans([trace("1", 0, 10 * MINUTE)], "relative");
      expect(s.get("1")).toEqual({ lower: -0.5 * MINUTE, upper: 10.5 * MINUTE });
    });

    it("should pad a short window by at least the floor", () => {
      const s = spans([trace("1", 0, SECOND)], "relative");
      expect(s.get("1")).toEqual({ lower: -SECOND, upper: 2 * SECOND });
    });

    it("should not split windows of one group", () => {
      const s = spans([trace("1", 0, MINUTE), trace("2", 0, MINUTE)], "relative");
      expect(s.get("1")).toEqual(s.get("2"));
    });

    it("should leave the end of a growing window open, past a fixed margin", () => {
      const s = spans([trace("1", 0, MINUTE, { edge: "growing" })], "relative");
      expect(s.get("1")).toEqual({ lower: -SECOND, upper: Infinity });
    });
  });

  it("should give no spans on a linear axis", () => {
    expect(spans([trace("1", 0, 10)], "linear").size).toBe(0);
  });
});

describe("Layout.span", () => {
  it("should give a line with no span every value", () => {
    const layout = new Layout([{ lower: 0, upper: 1 }], box.DECIMAL);
    expect(layout.span("x")).toEqual(bounds.INFINITE);
  });
});

describe("collapse", () => {
  const MIN = MINUTE;

  it("should give one segment padded by a share of its span for one window", () => {
    expect(collapse([trace("1", 0, 100 * MIN)], 0.01)).toEqual([
      { lower: -MIN, upper: 101 * MIN },
    ]);
  });

  it("should give each window far from the others its own segment", () => {
    const s = collapse([trace("1", 0, 10 * MIN), trace("2", 100 * MIN, 110 * MIN)], 0);
    expect(s).toEqual([
      { lower: 0, upper: 10.5 * MIN },
      { lower: 99.5 * MIN, upper: 110 * MIN },
    ]);
  });

  it("should merge windows whose margins overlap", () => {
    const s = collapse([trace("1", 0, 10 * MIN), trace("2", 10.8 * MIN, 20 * MIN)], 0);
    expect(s).toEqual([{ lower: 0, upper: 20 * MIN }]);
  });

  it("should pad the outer edges by a share of the span of every segment", () => {
    const s = collapse(
      [trace("1", 0, 10 * MIN), trace("2", 100 * MIN, 110 * MIN)],
      0.1,
    );
    expect(s[0].lower).toBeCloseTo(-2.1 * MIN);
    expect(s[1].upper).toBeCloseTo(112.1 * MIN);
  });

  it("should give a growing window the floor before it and no margin after", () => {
    const s = collapse(
      [trace("1", 0, 10 * MIN), trace("2", 100 * MIN, 110 * MIN, { edge: "growing" })],
      0,
    );
    expect(s[1]).toEqual({ lower: 100 * MIN - SECOND, upper: 110 * MIN });
  });

  it("should give a rolling window a margin from its span before it", () => {
    const s = collapse(
      [trace("1", 0, 10 * MIN), trace("2", 100 * MIN, 110 * MIN, { edge: "rolling" })],
      0,
    );
    expect(s[1]).toEqual({ lower: 99.5 * MIN, upper: 110 * MIN });
  });

  it("should pad a single sample by one on each side", () => {
    expect(collapse([trace("1", 5, 5)], 0.01)).toEqual([{ lower: 4, upper: 6 }]);
  });

  it("should give no segments when no trace has a window", () => {
    expect(collapse([trace("1", Infinity, -Infinity)], 0.01)).toEqual([]);
  });
});

describe("Layout.segment", () => {
  const layout = new Layout(
    [
      { lower: 0, upper: 10 },
      { lower: 100, upper: 110 },
    ],
    box.DECIMAL,
    new Map([
      ["1", { lower: -Infinity, upper: 10 }],
      ["2", { lower: 100, upper: Infinity }],
    ]),
  );

  it("should give the segment of the span of a line", () => {
    expect(layout.segment("1")).toBe(layout.segments[0]);
    expect(layout.segment("2")).toBe(layout.segments[1]);
  });

  it("should give the first segment for a line with no span", () => {
    expect(layout.segment("3")).toBe(layout.segments[0]);
  });

  it("should extend the positions of the outer segments without end", () => {
    expect(layout.segments[0].position).toEqual({ lower: -Infinity, upper: 0.5 });
    expect(layout.segments[1].position).toEqual({ lower: 0.5, upper: Infinity });
  });
});

describe("join", () => {
  const PIECES = [
    { lower: 0, upper: 10 },
    { lower: 100, upper: 110 },
    { lower: 200, upper: 210 },
  ];

  it("should list a closed gap between each two pieces", () => {
    expect(join(PIECES, [])).toEqual({
      segments: PIECES,
      gaps: [
        { bounds: { lower: 10, upper: 100 }, open: false },
        { bounds: { lower: 110, upper: 200 }, open: false },
      ],
    });
  });

  it("should merge the pieces on each side of an open gap", () => {
    const { segments: s, gaps } = join(PIECES, [110]);
    expect(s).toEqual([
      { lower: 0, upper: 10 },
      { lower: 100, upper: 210 },
    ]);
    expect(gaps.map((g) => g.open)).toEqual([false, true]);
  });

  it("should give one segment when every gap is open", () => {
    expect(join(PIECES, [10, 110]).segments).toEqual([{ lower: 0, upper: 210 }]);
  });
});

describe("Layout.gapAt", () => {
  const PIECES = [
    { lower: 0, upper: 10 },
    { lower: 100, upper: 110 },
    { lower: 200, upper: 210 },
  ];
  const layout = (open: number[]) => {
    const { segments: s, gaps } = join(PIECES, open);
    return new Layout(s, box.DECIMAL, new Map(), gaps);
  };

  it("should find the closed gap whose seam is within tolerance", () => {
    expect(layout([]).gapAt(1 / 3 + 0.01, 0.02)?.bounds).toEqual({
      lower: 10,
      upper: 100,
    });
  });

  it("should find no gap outside the tolerance", () => {
    expect(layout([]).gapAt(0.4, 0.02)).toBeNull();
  });

  it("should mark an open gap at its middle", () => {
    const l = layout([110]);
    const open = l.gaps[1];
    expect(l.gapAt(l.mark(open), 0.01)).toBe(open);
    expect(l.pos(155)).toBeCloseTo(l.mark(open));
  });

  it("should skip an open gap when it finds only closed gaps", () => {
    const l = layout([110]);
    expect(l.gapAt(l.mark(l.gaps[1]), 0.01, true)).toBeNull();
  });
});

describe("Layout breaks", () => {
  const PIECES = [
    { lower: 0, upper: 10 },
    { lower: 100, upper: 110 },
    { lower: 200, upper: 210 },
  ];
  const layout = (open: number[]) => {
    const { segments: s, gaps } = join(PIECES, open);
    return new Layout(s, box.DECIMAL, new Map(), gaps);
  };

  it("should list the home position of each seam", () => {
    const [a, b] = layout([]).seams;
    expect(a).toBeCloseTo(1 / 3);
    expect(b).toBeCloseTo(2 / 3);
  });

  it("should list no seam for an open gap", () => {
    expect(layout([10]).seams).toHaveLength(1);
  });

  it("should find a closed gap between two values in either order", () => {
    expect(layout([]).closedBetween(5, 105)).toBe(true);
    expect(layout([]).closedBetween(105, 5)).toBe(true);
  });

  it("should find no closed gap inside one segment or across an open gap", () => {
    expect(layout([]).closedBetween(1, 9)).toBe(false);
    expect(layout([10]).closedBetween(5, 105)).toBe(false);
  });
});
