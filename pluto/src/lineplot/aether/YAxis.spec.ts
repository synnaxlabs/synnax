// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  bounds,
  box,
  color,
  DataType,
  MultiSeries,
  scale,
  Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { afterEach, describe, expect, it, vi } from "vitest";

import { aetherTest } from "@/aether/testutil";
import { XAxis } from "@/lineplot/aether/XAxis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { buildStack } from "@/testutil/providers";
import { Line } from "@/vis/line/aether/line";
import { lineTest } from "@/vis/line/aether/testutil";
import { canvasTest } from "@/vis/render/testutil";

// Covers the calls the line programs make when they compile.
const GL = {
  createProgram: () => ({}),
  createShader: () => ({}),
  shaderSource: () => {},
  compileShader: () => {},
  getShaderParameter: () => true,
  attachShader: () => {},
  linkProgram: () => {},
};

const seriesValue = (data: number[]): [bounds.Bounds, MultiSeries] => {
  const s = new Series({
    data: new Float32Array(data),
    dataType: DataType.FLOAT32,
    timeRange: new TimeRange(0, 100),
  });
  return [s.bounds, new MultiSeries([s])];
};

const specOf = (
  source: telemTest.TestSource<[bounds.Bounds, MultiSeries]>,
): telem.SeriesSourceSpec => ({
  type: telemTest.TestSource.TYPE,
  props: { testId: source.id },
  variant: "source",
  valueType: "series",
});

const sourceSpec = (data: number[]): telem.SeriesSourceSpec =>
  specOf(telemTest.source(seriesValue(data)));

interface MountProps {
  lines?: Record<string, Record<string, unknown>>;
  hidden?: string[];
  axis?: Record<string, unknown>;
  xAxis?: Record<string, unknown>;
}

interface Mount {
  xAxis: XAxis;
  axis: YAxis;
  setVisible: (line: string, visible: boolean) => void;
  calls: canvasTest.RecordingGL["calls"];
}

describe("YAxis", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    vi.useRealTimers();
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  const mount = ({
    lines = {
      nominal: { x: sourceSpec([0, 5, 10]), y: sourceSpec([1, 2, 3]) },
      railed: { x: sourceSpec([0, 50]), y: sourceSpec([100, 1]) },
    },
    hidden = [],
    axis = {},
    xAxis = {},
  }: MountProps = {}): Mount => {
    const recorder = canvasTest.record();
    const { gl, calls } = canvasTest.createGL();
    (recorder as { gl: unknown }).gl = gl;
    const stack = buildStack({
      registry: {
        [lineTest.Host.TYPE]: lineTest.Host,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [Line.TYPE]: Line,
      },
      render: recorder,
    });
    const xPath = [...stack.basePath, "host", "x1"];
    const path = [...xPath, "y1"];
    stack.driver.update(xPath.slice(0, -1), lineTest.Host.TYPE, {});
    stack.driver.update(xPath, XAxis.TYPE, {
      location: "bottom",
      autoBoundPadding: 0,
      ...xAxis,
    });
    stack.driver.update(path, YAxis.TYPE, {
      location: "left",
      autoBoundPadding: 0,
      ...axis,
    });
    const setVisible = (line: string, visible: boolean): void =>
      stack.driver.update([...path, line], Line.TYPE, {
        color: color.construct("#ff0000"),
        ...lines[line],
        visible,
      });
    Object.keys(lines).forEach((line) => setVisible(line, !hidden.includes(line)));
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    return {
      xAxis: stack.driver.find<XAxis>(xPath),
      axis: stack.driver.find<YAxis>(path),
      setVisible,
      calls,
    };
  };

  describe("render", () => {
    it("should draw a raised line over the other lines", () => {
      const RAISED = color.construct("#00ff00");
      const m = mount({
        lines: {
          raised: {
            x: sourceSpec([0, 50]),
            y: sourceSpec([1, 2]),
            color: RAISED,
            raised: true,
          },
          plain: { x: sourceSpec([0, 50]), y: sourceSpec([3, 4]) },
        },
      });
      const region = box.construct(0, 0, 800, 600);
      m.axis.render({
        grid: {},
        plot: region,
        viewport: box.DECIMAL,
        container: region,
        canvases: ["gl"],
        hold: false,
        xDataToDecimalScale: scale.Scale.scale<number>(0, 50).scale(0, 1),
        xBounds: { lower: 0, upper: 50 },
      });
      const colors = m.calls
        .filter((c) => c.op === "uniform4fv")
        .map((c) => [...(c.args[1] as Iterable<number>)]);
      expect(colors.length).toBeGreaterThan(1);
      expect(colors.at(-1)).toEqual(color.rgba1(RAISED));
      expect(colors[0]).not.toEqual(color.rgba1(RAISED));
    });
  });

  describe("autoBoundPadding", () => {
    it("should default to 10% on a y axis and 1% on an x axis", () => {
      const m = mount({
        axis: { autoBoundPadding: undefined },
        xAxis: { autoBoundPadding: undefined },
      });
      expect(m.axis.state.autoBoundPadding).toBe(0.1);
      expect(m.xAxis.state.autoBoundPadding).toBe(0.01);
    });
  });

  describe("bounds", () => {
    it("should cover every visible line", () => {
      const m = mount();
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual({ lower: 1, upper: 100 });
    });

    it("should exclude a hidden line", () => {
      const m = mount();
      m.setVisible("railed", false);
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual({ lower: 1, upper: 3 });
    });

    it("should exclude a hidden line inside a finite x window", () => {
      const m = mount();
      const xWindow = { lower: 0, upper: 5 };
      expect(m.axis.bounds(false, xWindow)).toEqual({ lower: 1, upper: 100 });
      m.setVisible("railed", false);
      expect(m.axis.bounds(false, xWindow)).toEqual({ lower: 1, upper: 2 });
    });

    it("should exclude a hidden line when the x window clips out every sample", () => {
      const m = mount();
      const xWindow = { lower: 20, upper: 30 };
      expect(m.axis.bounds(false, xWindow)).toEqual({ lower: 1, upper: 100 });
      m.setVisible("railed", false);
      expect(m.axis.bounds(false, xWindow)).toEqual({ lower: 1, upper: 3 });
    });

    it("should use the x axis bounds of the visible lines as its window", () => {
      const m = mount();
      m.setVisible("railed", false);
      const xBounds = m.xAxis.bounds(false);
      expect(xBounds).toEqual({ lower: 0, upper: 10 });
      expect(m.axis.bounds(false, xBounds)).toEqual({ lower: 1, upper: 3 });
    });

    it("should include a line again when it is shown", () => {
      const m = mount();
      m.setVisible("railed", false);
      m.setVisible("railed", true);
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual({ lower: 1, upper: 100 });
    });

    it("should fall back to decimal bounds when every line is hidden", () => {
      const m = mount();
      m.setVisible("nominal", false);
      m.setVisible("railed", false);
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual(bounds.DECIMAL);
    });

    it("should pad the data bounds by the auto bound padding", () => {
      const m = mount({
        lines: { line: { x: sourceSpec([0, 1]), y: sourceSpec([0, 10]) } },
        axis: { autoBoundPadding: 0.1 },
      });
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual({ lower: -1, upper: 11 });
    });

    it("should expand equal data bounds by one", () => {
      const m = mount({
        lines: { line: { x: sourceSpec([0, 1]), y: sourceSpec([5, 5]) } },
      });
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual({ lower: 4, upper: 6 });
    });

    it("should ignore a line without data", () => {
      const m = mount({
        lines: {
          empty: { x: sourceSpec([]), y: sourceSpec([]) },
          line: { x: sourceSpec([0, 1]), y: sourceSpec([0, 10]) },
        },
      });
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual({ lower: 0, upper: 10 });
    });

    it("should fall back to an hour from now on an empty time axis", () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2030-01-02T03:04:05Z"));
      const m = mount({ axis: { type: "time" }, hidden: ["nominal", "railed"] });
      const now = TimeStamp.now();
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual({
        lower: Number(now.valueOf()),
        upper: Number(now.add(TimeSpan.HOUR).valueOf()),
      });
    });

    it("should move the empty time axis with the clock", () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2030-01-02T03:04:05Z"));
      const m = mount({ axis: { type: "time" }, hidden: ["nominal", "railed"] });
      const first = m.axis.bounds(false, bounds.INFINITE);
      vi.setSystemTime(new Date("2030-01-02T04:04:05Z"));
      const second = m.axis.bounds(false, bounds.INFINITE);
      expect(second.lower).toBeGreaterThan(first.lower);
      expect(second.lower).toBe(Number(TimeStamp.now().valueOf()));
    });

    it("should exclude a line that is hidden when it mounts", () => {
      const m = mount({ hidden: ["railed"] });
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual({ lower: 1, upper: 3 });
    });

    it("should keep the held bounds when a line is hidden during a hold", () => {
      const m = mount();
      m.axis.bounds(false, bounds.INFINITE);
      m.setVisible("railed", false);
      expect(m.axis.bounds(true, bounds.INFINITE)).toEqual({ lower: 1, upper: 100 });
    });

    it("should exclude a hidden line when the hold ends", () => {
      const m = mount();
      m.axis.bounds(false, bounds.INFINITE);
      m.setVisible("railed", false);
      m.axis.bounds(true, bounds.INFINITE);
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual({ lower: 1, upper: 3 });
    });

    it("should hold bounds only once its lines load", () => {
      const y = telemTest.source(seriesValue([1, 2, 3]));
      y.status = telem.LOADING_STATUS;
      const m = mount({
        lines: { loading: { x: sourceSpec([0, 5, 10]), y: specOf(y) } },
      });
      expect(m.axis.bounds(true, bounds.INFINITE)).toEqual({ lower: 1, upper: 3 });
      y.status = telem.SUCCESS_STATUS;
      y.setValue(seriesValue([1, 200]));
      expect(m.axis.bounds(true, bounds.INFINITE)).toEqual({ lower: 1, upper: 200 });
      y.setValue(seriesValue([1, 2]));
      expect(m.axis.bounds(true, bounds.INFINITE)).toEqual({ lower: 1, upper: 200 });
    });

    it("should change only the automatic bound of a partly manual axis", () => {
      const m = mount({
        axis: { bounds: { lower: -5, upper: 500 }, manualBounds: { lower: true } },
      });
      m.setVisible("railed", false);
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual({ lower: -5, upper: 3 });
    });
  });

  describe("xBounds", () => {
    it("should return the x bounds of every visible line", () => {
      const m = mount();
      expect(m.axis.xBounds().map((b) => b.bounds)).toEqual([
        { lower: 0, upper: 10 },
        { lower: 0, upper: 50 },
      ]);
    });

    it("should exclude a hidden line", () => {
      const m = mount();
      m.setVisible("railed", false);
      expect(m.axis.xBounds().map((b) => b.bounds)).toEqual([{ lower: 0, upper: 10 }]);
    });

    it("should return no bounds when every line is hidden", () => {
      const m = mount({ hidden: ["nominal", "railed"] });
      expect(m.axis.xBounds()).toEqual([]);
    });

    it("should report loading while a visible line's x source loads", () => {
      const x = telemTest.source(seriesValue([0, 5, 10]));
      x.status = telem.LOADING_STATUS;
      const m = mount({ lines: { loading: { x: specOf(x), y: sourceSpec([1, 2]) } } });
      expect(m.axis.xBounds()[0].loading).toBe(true);
    });
  });
});

describe("XAxis", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  // Mounts an x axis with a y axis per entry, each holding lines of the given x bounds.
  const mount = (
    yAxes: Record<string, Array<{ x: bounds.Bounds; edge?: telem.Edge }>>,
  ): XAxis => {
    const recorder = canvasTest.record();
    (recorder as { gl: unknown }).gl = GL;
    const stack = buildStack({
      registry: {
        [lineTest.Host.TYPE]: lineTest.Host,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [Line.TYPE]: Line,
      },
      render: recorder,
    });
    const xPath = [...stack.basePath, "host", "x1"];
    stack.driver.update(xPath.slice(0, -1), lineTest.Host.TYPE, {});
    stack.driver.update(xPath, XAxis.TYPE, { location: "bottom", autoBoundPadding: 0 });
    Object.entries(yAxes).forEach(([yKey, lines]) => {
      const yPath = [...xPath, yKey];
      stack.driver.update(yPath, YAxis.TYPE, { location: "left", autoBoundPadding: 0 });
      lines.forEach(({ x: b, edge }, i) => {
        const x = telemTest.source<[bounds.Bounds, MultiSeries]>([
          b,
          new MultiSeries(),
        ]);
        x.edge = edge;
        stack.driver.update([...yPath, `line${i}`], Line.TYPE, {
          x: specOf(x),
          y: sourceSpec([1, 2]),
          color: color.construct("#ff0000"),
          visible: true,
        });
      });
    });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    return stack.driver.find<XAxis>(xPath);
  };

  describe("bounds", () => {
    it("should cover the x bounds of every line", () => {
      const x = mount({
        y1: [{ x: { lower: 0, upper: 10 } }],
        y2: [{ x: { lower: 5, upper: 50 } }],
      });
      expect(x.bounds(false)).toEqual({ lower: 0, upper: 50 });
    });

    it("should end every rolling line at the newest edge across y axes", () => {
      const x = mount({
        y1: [{ x: { lower: 0, upper: 30 }, edge: "rolling" }],
        y2: [{ x: { lower: 3570, upper: 3600 }, edge: "rolling" }],
      });
      expect(x.bounds(false)).toEqual({ lower: 3570, upper: 3600 });
    });

    it("should keep the longest span of the rolling lines", () => {
      const x = mount({
        y1: [
          { x: { lower: 0, upper: 300 }, edge: "rolling" },
          { x: { lower: 970, upper: 1000 }, edge: "rolling" },
        ],
      });
      expect(x.bounds(false)).toEqual({ lower: 700, upper: 1000 });
    });

    it("should leave the bounds of a line that does not roll in place", () => {
      const x = mount({
        y1: [
          { x: { lower: 0, upper: 30 }, edge: "rolling" },
          { x: { lower: 1000, upper: 1030 }, edge: "rolling" },
          { x: { lower: 10, upper: 20 } },
        ],
      });
      expect(x.bounds(false)).toEqual({ lower: 10, upper: 1030 });
    });

    it("should end a growing line at the newest edge and keep its start", () => {
      const x = mount({
        y1: [
          { x: { lower: 100, upper: 900 }, edge: "growing" },
          { x: { lower: 970, upper: 1000 }, edge: "rolling" },
        ],
      });
      expect(x.bounds(false)).toEqual({ lower: 100, upper: 1000 });
    });

    it("should end a rolling line at the edge of a growing line", () => {
      const x = mount({
        y1: [{ x: { lower: 0, upper: 30 }, edge: "rolling" }],
        y2: [{ x: { lower: 100, upper: 2000 }, edge: "growing" }],
      });
      expect(x.bounds(false)).toEqual({ lower: 100, upper: 2000 });
    });

    it("should ignore a rolling line with no bounds", () => {
      const x = mount({
        y1: [
          { x: bounds.INVALID, edge: "rolling" },
          { x: { lower: 970, upper: 1000 }, edge: "rolling" },
        ],
      });
      expect(x.bounds(false)).toEqual({ lower: 970, upper: 1000 });
    });
  });
});
