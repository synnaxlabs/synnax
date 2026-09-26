// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box } from "@synnaxlabs/x";
import { afterEach, describe, expect, it } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { aetherTest } from "@/aether/test";
import { LinePlot } from "@/lineplot/aether/LinePlot";
import { XAxis } from "@/lineplot/aether/XAxis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { buildStack } from "@/testutil/providers";
import { Line } from "@/vis/line/aether/line";
import { type render } from "@/vis/render";
import { canvasTest } from "@/vis/render/test";

const stubLineStateZ = z.object({ loading: z.boolean() });

// Line needs a live GL context, so a stub with its TYPE stands in for the walk.
class StubLine extends aether.Leaf<typeof stubLineStateZ> {
  static readonly TYPE = Line.TYPE;
  schema = stubLineStateZ;

  get loading(): boolean {
    return this.state.loading;
  }
}

interface LoopEntry {
  key?: string;
  render?: () => render.Cleanup | undefined;
}

interface Mount {
  plot: LinePlot;
  recorder: canvasTest.Recorder;
  setLineLoading: (loading: boolean) => void;
  pump: () => void;
}

describe("LinePlot", () => {
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  const mount = (lineLoading: boolean | null): Mount => {
    const recorder = new canvasTest.Recorder();
    const stack = buildStack({
      registry: {
        [LinePlot.TYPE]: LinePlot,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [StubLine.TYPE]: StubLine,
      },
      render: recorder,
    });
    const base = [...stack.basePath, "plot"];
    stack.driver.update(base, LinePlot.TYPE, {
      container: box.DECIMAL,
      viewport: box.DECIMAL,
      grid: {},
    });
    stack.driver.update([...base, "x1"], XAxis.TYPE, { location: "bottom" });
    stack.driver.update([...base, "x1", "y1"], YAxis.TYPE, { location: "left" });
    if (lineLoading != null)
      stack.driver.update([...base, "x1", "y1", "l1"], StubLine.TYPE, {
        loading: lineLoading,
      });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    const pump = (): void => {
      const entry = recorder.loopCalls
        .map((call) => call.args[0] as LoopEntry)
        .reverse()
        .find((e) => e.key?.startsWith(LinePlot.TYPE) ?? false);
      entry?.render?.();
    };
    return {
      plot: stack.driver.find<LinePlot>(base),
      recorder,
      setLineLoading: (loading) =>
        stack.driver.update([...base, "x1", "y1", "l1"], StubLine.TYPE, { loading }),
      pump,
    };
  };

  describe("loading", () => {
    it("should suppress draws and sync state while a line back-fills", () => {
      const m = mount(true);
      m.pump();
      expect(m.plot.state.loading).toBe(true);
      expect(m.recorder.scissorCalls).toHaveLength(0);
    });

    it("should resume draws once every line settles", () => {
      const m = mount(true);
      m.pump();
      m.setLineLoading(false);
      m.pump();
      expect(m.plot.state.loading).toBe(false);
      expect(m.recorder.scissorCalls.length).toBeGreaterThan(0);
    });

    it("should not report loading for a plot with no lines", () => {
      const m = mount(null);
      m.pump();
      expect(m.plot.state.loading).toBe(false);
    });
  });
});
