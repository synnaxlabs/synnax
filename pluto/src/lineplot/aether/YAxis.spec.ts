// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, color, DataType, MultiSeries, Series, TimeRange } from "@synnaxlabs/x";
import { afterEach, describe, expect, it } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { aetherTest } from "@/aether/test";
import { XAxis } from "@/lineplot/aether/XAxis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { type telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/test";
import { buildStack } from "@/testutil/providers";
import { Context, Line } from "@/vis/line/aether/line";
import { render } from "@/vis/render";
import { canvasTest } from "@/vis/render/test";

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

// Supplies the line GL context and the render requestor, as the canvas and plot do.
class Host extends aether.Composite<typeof Host.stateZ> {
  static readonly TYPE = "y-axis-test-host";
  static readonly stateZ = z.object({});
  schema = Host.stateZ;

  afterUpdate(ctx: aether.Context): void {
    Context.create(ctx, render.Context.use(ctx));
    render.control(ctx, () => {});
  }
}

const sourceSpec = (data: number[]): telem.SeriesSourceSpec => {
  const s = new Series({
    data: new Float32Array(data),
    dataType: DataType.FLOAT32,
    timeRange: new TimeRange(0, 100),
  });
  return {
    type: telemTest.TestSource.TYPE,
    props: { testId: telemTest.source([s.bounds, new MultiSeries([s])]).id },
    variant: "source",
    valueType: "series",
  };
};

interface MountProps {
  hidden?: string[];
  axis?: Record<string, unknown>;
}

interface Mount {
  xAxis: XAxis;
  axis: YAxis;
  setVisible: (line: string, visible: boolean) => void;
}

describe("YAxis", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  const mount = ({ hidden = [], axis = {} }: MountProps = {}): Mount => {
    const recorder = canvasTest.record();
    (recorder as { gl: unknown }).gl = GL;
    const stack = buildStack({
      registry: {
        [Host.TYPE]: Host,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [Line.TYPE]: Line,
      },
      render: recorder,
    });
    const xPath = [...stack.basePath, "host", "x1"];
    const path = [...xPath, "y1"];
    stack.driver.update(xPath.slice(0, -1), Host.TYPE, {});
    stack.driver.update(xPath, XAxis.TYPE, { location: "bottom", autoBoundPadding: 0 });
    stack.driver.update(path, YAxis.TYPE, {
      location: "left",
      autoBoundPadding: 0,
      ...axis,
    });
    const lines: Record<string, Record<string, unknown>> = {
      nominal: { x: sourceSpec([0, 5, 10]), y: sourceSpec([1, 2, 3]) },
      railed: { x: sourceSpec([0, 50]), y: sourceSpec([100, 1]) },
    };
    const setVisible = (line: string, visible: boolean): void =>
      stack.driver.update([...path, line], Line.TYPE, {
        ...lines[line],
        color: color.construct("#ff0000"),
        visible,
      });
    Object.keys(lines).forEach((line) => setVisible(line, !hidden.includes(line)));
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    return {
      xAxis: stack.driver.find<XAxis>(xPath),
      axis: stack.driver.find<YAxis>(path),
      setVisible,
    };
  };

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

    it("should change only the automatic bound of a partly manual axis", () => {
      const m = mount({
        axis: { bounds: { lower: -5, upper: 500 }, manualBounds: { lower: true } },
      });
      m.setVisible("railed", false);
      expect(m.axis.bounds(false, bounds.INFINITE)).toEqual({ lower: -5, upper: 3 });
    });
  });

  describe("xBounds", () => {
    it("should cover every visible line", () => {
      const m = mount();
      expect(m.axis.xBounds()).toEqual({ lower: 0, upper: 50 });
    });

    it("should exclude a hidden line", () => {
      const m = mount();
      m.setVisible("railed", false);
      expect(m.axis.xBounds()).toEqual({ lower: 0, upper: 10 });
    });

    it("should return invalid bounds when every line is hidden", () => {
      const m = mount({ hidden: ["nominal", "railed"] });
      expect(m.axis.xBounds()).toEqual(bounds.INVALID);
    });
  });
});
