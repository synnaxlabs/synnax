// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { lineplot } from "@/lineplot";

const createEmpty = (overrides: Partial<lineplot.LinePlot> = {}): lineplot.LinePlot =>
  lineplot.newZ.parse({ name: "", ...overrides });

const apply = (
  state: lineplot.LinePlot,
  ...actions: lineplot.Action[]
): lineplot.LinePlot => lineplot.reduceAll(state, actions).next;

const roundTrip = (
  state: lineplot.LinePlot,
  ...actions: lineplot.Action[]
): lineplot.LinePlot => {
  const { next, inverse } = lineplot.reduceAll(state, actions);
  return lineplot.reduceAll(next, inverse).next;
};

const createLine = (key: string, color: string): lineplot.Line =>
  lineplot.lineZ.parse({ key, color });

const createRule = (key: string, label: string): lineplot.Rule =>
  lineplot.ruleZ.parse({
    key,
    label,
    color: color.construct("#888888"),
    axis: "y1",
    position: 0,
  });

describe("lineplot reducer", () => {
  describe("rename", () => {
    it("should set the line plot name", () => {
      const out = apply(createEmpty({ name: "old" }), lineplot.rename({ name: "new" }));
      expect(out.name).toEqual("new");
    });
    it("should round-trip the previous name through its inverse", () => {
      const state = createEmpty({ name: "before" });
      expect(roundTrip(state, lineplot.rename({ name: "after" })).name).toEqual(
        "before",
      );
    });
    it("should target the plot key so plot-level edits invalidate each other", () => {
      const state = createEmpty({ key: "11111111-1111-4111-8111-111111111111" });
      const { targets } = lineplot.reduceAll(state, [lineplot.rename({ name: "x" })]);
      expect(targets).toEqual([state.key]);
    });
  });

  describe("setTitle", () => {
    it("should replace the title configuration", () => {
      const out = apply(
        createEmpty(),
        lineplot.setTitle({ title: { level: "h2", visible: true } }),
      );
      expect(out.title).toEqual({ level: "h2", visible: true });
    });
    it("should round-trip the previous title through its inverse", () => {
      const state = createEmpty({ title: { level: "h4", visible: false } });
      expect(
        roundTrip(state, lineplot.setTitle({ title: { level: "h1", visible: true } }))
          .title,
      ).toEqual(state.title);
    });
  });

  describe("setLegendHidden", () => {
    it("should set legend visibility, leaving position unchanged", () => {
      const out = apply(createEmpty(), lineplot.setLegendHidden({ hidden: true }));
      expect(out.legend.hidden).toEqual(true);
      expect(out.legend.position).toEqual(createEmpty().legend.position);
    });
    it("should round-trip the previous visibility through its inverse", () => {
      const state = createEmpty();
      expect(
        roundTrip(state, lineplot.setLegendHidden({ hidden: true })).legend,
      ).toEqual(state.legend);
    });
  });

  describe("setLegendPosition", () => {
    it("should set legend position, leaving visibility unchanged", () => {
      const out = apply(
        createEmpty(),
        lineplot.setLegendPosition({
          position: {
            x: 10,
            y: 20,
            root: { x: "left", y: "top" },
            units: { x: "px", y: "px" },
          },
        }),
      );
      expect(out.legend.position.x).toEqual(10);
      expect(out.legend.hidden).toEqual(createEmpty().legend.hidden);
    });
    it("should round-trip the previous position through its inverse", () => {
      const state = createEmpty();
      expect(
        roundTrip(
          state,
          lineplot.setLegendPosition({
            position: {
              x: 1,
              y: 2,
              root: { x: "left", y: "top" },
              units: { x: "px", y: "px" },
            },
          }),
        ).legend,
      ).toEqual(state.legend);
    });
  });

  describe("addChannel", () => {
    it("should append the channel to the named y-axis", () => {
      const out = apply(
        createEmpty(),
        lineplot.addChannel({ axisKey: "y1", channel: 42 }),
      );
      expect(out.channels.y1).toEqual([42]);
    });
    it("should be a no-op when the channel is already present", () => {
      const state = createEmpty({
        channels: { x1: 0, x2: 0, y1: [42], y2: [], y3: [], y4: [] },
      });
      const out = apply(state, lineplot.addChannel({ axisKey: "y1", channel: 42 }));
      expect(out.channels.y1).toEqual([42]);
    });
    it("should round-trip through removeChannel as its inverse", () => {
      const state = createEmpty();
      expect(
        roundTrip(state, lineplot.addChannel({ axisKey: "y2", channel: 7 })).channels
          .y2,
      ).toEqual([]);
    });
    it("should target the channel so distinct channels do not invalidate each other", () => {
      const { targets } = lineplot.reduceAll(createEmpty(), [
        lineplot.addChannel({ axisKey: "y1", channel: 42 }),
      ]);
      expect(targets).toEqual(["channel:42"]);
    });
  });

  describe("removeChannel", () => {
    it("should drop the channel from the named y-axis", () => {
      const state = createEmpty({
        channels: { x1: 0, x2: 0, y1: [1, 2, 3], y2: [], y3: [], y4: [] },
      });
      const out = apply(state, lineplot.removeChannel({ axisKey: "y1", channel: 2 }));
      expect(out.channels.y1).toEqual([1, 3]);
    });
    it("should be a no-op when the channel is not present", () => {
      const state = createEmpty({
        channels: { x1: 0, x2: 0, y1: [1], y2: [], y3: [], y4: [] },
      });
      const out = apply(state, lineplot.removeChannel({ axisKey: "y1", channel: 99 }));
      expect(out.channels.y1).toEqual([1]);
    });
    it("should round-trip through addChannel as its inverse", () => {
      const state = createEmpty({
        channels: { x1: 0, x2: 0, y1: [5], y2: [], y3: [], y4: [] },
      });
      expect(
        roundTrip(state, lineplot.removeChannel({ axisKey: "y1", channel: 5 })).channels
          .y1,
      ).toEqual([5]);
    });
  });

  describe("setXChannel", () => {
    it("should set the channel on the named x-axis", () => {
      const out = apply(
        createEmpty(),
        lineplot.setXChannel({ axisKey: "x1", channel: 99 }),
      );
      expect(out.channels.x1).toEqual(99);
    });
    it("should round-trip the previous x-channel through its inverse", () => {
      const state = createEmpty({
        channels: { x1: 42, x2: 0, y1: [], y2: [], y3: [], y4: [] },
      });
      expect(
        roundTrip(state, lineplot.setXChannel({ axisKey: "x1", channel: 99 })).channels
          .x1,
      ).toEqual(42);
    });
    it("should target the x-axis so x1 and x2 edits are independent", () => {
      const { targets } = lineplot.reduceAll(createEmpty(), [
        lineplot.setXChannel({ axisKey: "x2", channel: 99 }),
      ]);
      expect(targets).toEqual(["axis:x2"]);
    });
  });

  describe("addRange and removeRange", () => {
    const r1 = "00000000-0000-0000-0000-000000000001";
    const r2 = "00000000-0000-0000-0000-000000000002";

    it("should append the range to the named x-axis", () => {
      const out = apply(createEmpty(), lineplot.addRange({ axisKey: "x1", range: r1 }));
      expect(out.ranges.x1).toEqual([r1]);
    });
    it("should drop the range from the named x-axis", () => {
      const state = createEmpty({ ranges: { x1: [r1, r2], x2: [] } });
      const out = apply(state, lineplot.removeRange({ axisKey: "x1", range: r1 }));
      expect(out.ranges.x1).toEqual([r2]);
    });
    it("should round-trip add then remove via the captured inverses", () => {
      const state = createEmpty();
      expect(
        roundTrip(state, lineplot.addRange({ axisKey: "x2", range: r1 })).ranges.x2,
      ).toEqual([]);
    });
    it("should target the range so distinct ranges do not invalidate each other", () => {
      const { targets } = lineplot.reduceAll(createEmpty(), [
        lineplot.addRange({ axisKey: "x1", range: r1 }),
      ]);
      expect(targets).toEqual([`range:${r1}`]);
    });
  });

  describe("setChannels", () => {
    const r1 = "00000000-0000-0000-0000-000000000001";

    it("should replace the whole channel set on the named y-axis", () => {
      const state = createEmpty({
        channels: { x1: 0, x2: 0, y1: [1, 2, 3], y2: [], y3: [], y4: [] },
      });
      const out = apply(
        state,
        lineplot.setChannels({ axisKey: "y1", channels: [2, 4] }),
      );
      expect(out.channels.y1).toEqual([2, 4]);
    });

    it("should be a no-op when the set is unchanged", () => {
      const state = createEmpty({
        channels: { x1: 0, x2: 0, y1: [1, 2], y2: [], y3: [], y4: [] },
      });
      const { next, inverse, targets } = lineplot.reduceAll(state, [
        lineplot.setChannels({ axisKey: "y1", channels: [1, 2] }),
      ]);
      expect(next.channels.y1).toEqual([1, 2]);
      expect(inverse).toEqual([]);
      expect(targets).toEqual([]);
    });

    it("should target only the channels that changed", () => {
      const state = createEmpty({
        channels: { x1: 0, x2: 0, y1: [1, 2], y2: [], y3: [], y4: [] },
      });
      const { targets } = lineplot.reduceAll(state, [
        lineplot.setChannels({ axisKey: "y1", channels: [2, 3] }),
      ]);
      expect(targets).toEqual(["channel:1", "channel:3"]);
    });

    it("should round-trip the previous channel set through its inverse", () => {
      const state = createEmpty({
        channels: { x1: 0, x2: 0, y1: [1, 2], y2: [], y3: [], y4: [] },
      });
      expect(
        roundTrip(state, lineplot.setChannels({ axisKey: "y1", channels: [3, 4] }))
          .channels.y1,
      ).toEqual([1, 2]);
    });

    it("should drop lines for removed channels and add lines for new ones", () => {
      const state = createEmpty({
        channels: { x1: 10, x2: 0, y1: [5], y2: [], y3: [], y4: [] },
        ranges: { x1: [], x2: [] },
      });
      const withLines = apply(state, lineplot.addRange({ axisKey: "x1", range: r1 }));
      const out = apply(
        withLines,
        lineplot.setChannels({ axisKey: "y1", channels: [6] }),
      );
      expect(out.lines.map((l) => l.key)).toEqual([
        lineplot.lineKey({
          yAxis: "y1",
          xAxis: "x1",
          range: r1,
          xChannel: 10,
          yChannel: 6,
        }),
      ]);
    });

    it("should preserve styling of surviving channels and restore dropped lines on undo", () => {
      const state = createEmpty({
        channels: { x1: 10, x2: 0, y1: [5, 6], y2: [], y3: [], y4: [] },
        ranges: { x1: [], x2: [] },
      });
      let s = apply(state, lineplot.addRange({ axisKey: "x1", range: r1 }));
      const keep = lineplot.lineKey({
        yAxis: "y1",
        xAxis: "x1",
        range: r1,
        xChannel: 10,
        yChannel: 5,
      });
      const drop = lineplot.lineKey({
        yAxis: "y1",
        xAxis: "x1",
        range: r1,
        xChannel: 10,
        yChannel: 6,
      });
      s = apply(
        s,
        lineplot.setLineColor({ key: keep, color: color.construct("#ff0000") }),
      );
      s = apply(
        s,
        lineplot.setLineColor({ key: drop, color: color.construct("#00ff00") }),
      );
      const out = apply(s, lineplot.setChannels({ axisKey: "y1", channels: [5] }));
      const survivor = out.lines.find((l) => l.key === keep);
      expect(survivor?.color).toEqual(color.construct("#ff0000"));
      const restored = roundTrip(
        s,
        lineplot.setChannels({ axisKey: "y1", channels: [5] }),
      );
      const dropped = restored.lines.find((l) => l.key === drop);
      expect(dropped?.color).toEqual(color.construct("#00ff00"));
    });
  });

  describe("setRanges", () => {
    const r1 = "00000000-0000-0000-0000-000000000001";
    const r2 = "00000000-0000-0000-0000-000000000002";
    const r3 = "00000000-0000-0000-0000-000000000003";

    it("should replace the whole range set on the named x-axis", () => {
      const state = createEmpty({ ranges: { x1: [r1, r2], x2: [] } });
      const out = apply(state, lineplot.setRanges({ axisKey: "x1", ranges: [r2, r3] }));
      expect(out.ranges.x1).toEqual([r2, r3]);
    });

    it("should be a no-op when the set is unchanged", () => {
      const state = createEmpty({ ranges: { x1: [r1, r2], x2: [] } });
      const { next, inverse, targets } = lineplot.reduceAll(state, [
        lineplot.setRanges({ axisKey: "x1", ranges: [r1, r2] }),
      ]);
      expect(next.ranges.x1).toEqual([r1, r2]);
      expect(inverse).toEqual([]);
      expect(targets).toEqual([]);
    });

    it("should target only the ranges that changed", () => {
      const state = createEmpty({ ranges: { x1: [r1, r2], x2: [] } });
      const { targets } = lineplot.reduceAll(state, [
        lineplot.setRanges({ axisKey: "x1", ranges: [r2, r3] }),
      ]);
      expect(targets).toEqual([`range:${r1}`, `range:${r3}`]);
    });

    it("should round-trip the previous range set through its inverse", () => {
      const state = createEmpty({ ranges: { x1: [r1, r2], x2: [] } });
      expect(
        roundTrip(state, lineplot.setRanges({ axisKey: "x1", ranges: [r3] })).ranges.x1,
      ).toEqual([r1, r2]);
    });

    it("should drop lines for removed ranges and add lines for new ones", () => {
      const state = createEmpty({
        channels: { x1: 10, x2: 0, y1: [5], y2: [], y3: [], y4: [] },
        ranges: { x1: [], x2: [] },
      });
      const withLines = apply(state, lineplot.addRange({ axisKey: "x1", range: r1 }));
      expect(withLines.lines).toHaveLength(1);
      const out = apply(withLines, lineplot.setRanges({ axisKey: "x1", ranges: [r2] }));
      expect(out.lines.map((l) => l.key)).toEqual([
        lineplot.lineKey({
          yAxis: "y1",
          xAxis: "x1",
          range: r2,
          xChannel: 10,
          yChannel: 5,
        }),
      ]);
    });
  });

  describe("eager line creation", () => {
    const r1 = "00000000-0000-0000-0000-000000000001";
    const r2 = "00000000-0000-0000-0000-000000000002";

    it("should materialize a line per range when a y-channel is added", () => {
      const state = createEmpty({
        channels: { x1: 10, x2: 0, y1: [], y2: [], y3: [], y4: [] },
        ranges: { x1: [r1, r2], x2: [] },
      });
      const out = apply(state, lineplot.addChannel({ axisKey: "y1", channel: 5 }));
      expect(out.lines.map((l) => l.key)).toEqual([
        lineplot.lineKey({
          yAxis: "y1",
          xAxis: "x1",
          range: r1,
          xChannel: 10,
          yChannel: 5,
        }),
        lineplot.lineKey({
          yAxis: "y1",
          xAxis: "x1",
          range: r2,
          xChannel: 10,
          yChannel: 5,
        }),
      ]);
    });

    it("should give created lines the Oracle default styling", () => {
      const state = createEmpty({
        channels: { x1: 10, x2: 0, y1: [], y2: [], y3: [], y4: [] },
        ranges: { x1: [r1], x2: [] },
      });
      const out = apply(state, lineplot.addChannel({ axisKey: "y1", channel: 5 }));
      expect(out.lines[0]).toMatchObject({
        strokeWidth: 2,
        downsample: 1,
        downsampleMode: "decimate",
      });
    });

    it("should create no lines when there are no ranges (partial selection)", () => {
      const state = createEmpty({
        channels: { x1: 10, x2: 0, y1: [], y2: [], y3: [], y4: [] },
        ranges: { x1: [], x2: [] },
      });
      const out = apply(state, lineplot.addChannel({ axisKey: "y1", channel: 5 }));
      expect(out.channels.y1).toEqual([5]);
      expect(out.lines).toEqual([]);
    });

    it("should materialize a line per y-channel when a range is added", () => {
      const state = createEmpty({
        channels: { x1: 10, x2: 0, y1: [5, 6], y2: [], y3: [], y4: [] },
        ranges: { x1: [], x2: [] },
      });
      const out = apply(state, lineplot.addRange({ axisKey: "x1", range: r1 }));
      expect(out.lines.map((l) => l.key)).toEqual([
        lineplot.lineKey({
          yAxis: "y1",
          xAxis: "x1",
          range: r1,
          xChannel: 10,
          yChannel: 5,
        }),
        lineplot.lineKey({
          yAxis: "y1",
          xAxis: "x1",
          range: r1,
          xChannel: 10,
          yChannel: 6,
        }),
      ]);
    });

    it("should remove a channel's lines when it is removed", () => {
      const state = createEmpty({
        channels: { x1: 10, x2: 0, y1: [5, 6], y2: [], y3: [], y4: [] },
        ranges: { x1: [], x2: [] },
      });
      const withLines = apply(state, lineplot.addRange({ axisKey: "x1", range: r1 }));
      expect(withLines.lines).toHaveLength(2);
      const out = apply(
        withLines,
        lineplot.removeChannel({ axisKey: "y1", channel: 5 }),
      );
      expect(out.lines.map((l) => l.key)).toEqual([
        lineplot.lineKey({
          yAxis: "y1",
          xAxis: "x1",
          range: r1,
          xChannel: 10,
          yChannel: 6,
        }),
      ]);
    });

    it("should restore removed lines with their styling on undo", () => {
      const state = createEmpty({
        channels: { x1: 10, x2: 0, y1: [5], y2: [], y3: [], y4: [] },
        ranges: { x1: [], x2: [] },
      });
      const key = lineplot.lineKey({
        yAxis: "y1",
        xAxis: "x1",
        range: r1,
        xChannel: 10,
        yChannel: 5,
      });
      let s = apply(state, lineplot.addRange({ axisKey: "x1", range: r1 }));
      s = apply(
        s,
        lineplot.setLineColor({
          key: s.lines[0].key,
          color: color.construct("#ff0000"),
        }),
      );
      const restored = roundTrip(
        s,
        lineplot.removeChannel({ axisKey: "y1", channel: 5 }),
      );
      expect(restored.lines).toHaveLength(1);
      expect(restored.lines[0].key).toEqual(key);
      expect(restored.lines[0].color).toEqual(color.construct("#ff0000"));
    });

    it("should rekey a y-axis's lines when the x-channel changes", () => {
      const state = createEmpty({
        channels: { x1: 10, x2: 0, y1: [5], y2: [], y3: [], y4: [] },
        ranges: { x1: [], x2: [] },
      });
      const withLines = apply(state, lineplot.addRange({ axisKey: "x1", range: r1 }));
      const out = apply(
        withLines,
        lineplot.setXChannel({ axisKey: "x1", channel: 20 }),
      );
      expect(out.lines.map((l) => l.key)).toEqual([
        lineplot.lineKey({
          yAxis: "y1",
          xAxis: "x1",
          range: r1,
          xChannel: 20,
          yChannel: 5,
        }),
      ]);
    });
  });

  describe("fine-grained axis actions", () => {
    it.each([
      {
        label: "setAxisLabel",
        action: lineplot.setAxisLabel({ key: "y1", label: "pressure" }),
        expected: { label: "pressure" },
      },
      {
        label: "setAxisLabelDirection",
        action: lineplot.setAxisLabelDirection({ key: "y1", labelDirection: "x" }),
        expected: { labelDirection: "x" },
      },
      {
        label: "setAxisLabelLevel",
        action: lineplot.setAxisLabelLevel({ key: "y1", labelLevel: "h5" }),
        expected: { labelLevel: "h5" },
      },
      {
        label: "setAxisTickSpacing",
        action: lineplot.setAxisTickSpacing({ key: "y1", tickSpacing: 50 }),
        expected: { tickSpacing: 50 },
      },
    ])("$label sets its field and round-trips via inverse", ({ action, expected }) => {
      const state = createEmpty();
      expect(apply(state, action).axes.y1).toMatchObject(expected);
      expect(roundTrip(state, action).axes.y1).toEqual(state.axes.y1);
    });

    it("setAxisBounds sets bounds and manual flags together and round-trips", () => {
      const state = createEmpty();
      const action = lineplot.setAxisBounds({
        key: "y1",
        bounds: { lower: 1, upper: 2 },
        manualBounds: { lower: true, upper: true },
      });
      const out = apply(state, action).axes.y1;
      expect(out.bounds).toEqual({ lower: 1, upper: 2 });
      expect(out.manualBounds).toEqual({ lower: true, upper: true });
      expect(roundTrip(state, action).axes.y1).toEqual(state.axes.y1);
    });

    it("setAxisType sets the tick type, clears it with null, and round-trips", () => {
      const state = createEmpty({
        axes: { ...createEmpty().axes, x1: { ...createEmpty().axes.x1, type: "time" } },
      });
      expect(
        apply(state, lineplot.setAxisType({ key: "x1", type: "linear" })).axes.x1.type,
      ).toEqual("linear");
      expect(
        apply(state, lineplot.setAxisType({ key: "x1" })).axes.x1.type,
      ).toBeUndefined();
      expect(roundTrip(state, lineplot.setAxisType({ key: "x1" })).axes.x1).toEqual(
        state.axes.x1,
      );
    });

    it("targets the edited axis so distinct axes are independent", () => {
      const { targets } = lineplot.reduceAll(createEmpty(), [
        lineplot.setAxisLabel({ key: "y3", label: "x" }),
      ]);
      expect(targets).toEqual(["axis:y3"]);
    });
  });

  describe("fine-grained line actions", () => {
    const base = (): lineplot.LinePlot =>
      createEmpty({ lines: [createLine("l1", "#ff0000")] });
    it.each([
      {
        label: "setLineStrokeWidth",
        action: lineplot.setLineStrokeWidth({ key: "l1", strokeWidth: 5 }),
        expected: { strokeWidth: 5 },
      },
      {
        label: "setLineDownsample",
        action: lineplot.setLineDownsample({ key: "l1", downsample: 3 }),
        expected: { downsample: 3 },
      },
      {
        label: "setLineDownsampleMode",
        action: lineplot.setLineDownsampleMode({
          key: "l1",
          downsampleMode: "average",
        }),
        expected: { downsampleMode: "average" },
      },
      {
        label: "setLineColor",
        action: lineplot.setLineColor({ key: "l1", color: color.construct("#0000ff") }),
        expected: { color: color.construct("#0000ff") },
      },
      {
        label: "setLineLabel",
        action: lineplot.setLineLabel({ key: "l1", label: "flow" }),
        expected: { label: "flow" },
      },
    ])("$label sets its field and round-trips via inverse", ({ action, expected }) => {
      const state = base();
      expect(apply(state, action).lines[0]).toMatchObject(expected);
      expect(roundTrip(state, action).lines[0]).toEqual(state.lines[0]);
    });

    it("clears a nullable field with null and round-trips", () => {
      const state = base();
      expect(
        apply(state, lineplot.setLineColor({ key: "l1" })).lines[0].color,
      ).toBeUndefined();
      expect(roundTrip(state, lineplot.setLineColor({ key: "l1" })).lines[0]).toEqual(
        state.lines[0],
      );
    });

    it("restores a previously-unset field on undo", () => {
      const start: lineplot.Line = {
        key: "l1",
        strokeWidth: 2,
        downsample: 1,
        downsampleMode: "decimate",
      };
      const state = createEmpty({ lines: [start] });
      const action = lineplot.setLineColor({
        key: "l1",
        color: color.construct("#ff0000"),
      });
      expect(apply(state, action).lines[0].color).toEqual(color.construct("#ff0000"));
      expect(roundTrip(state, action).lines[0]).toEqual(state.lines[0]);
    });

    it("is a no-op when the line does not exist", () => {
      const { targets } = lineplot.reduceAll(createEmpty(), [
        lineplot.setLineColor({ key: "missing", color: color.construct("#ff0000") }),
      ]);
      expect(targets).toEqual([]);
    });

    it("targets the line so distinct lines are independent", () => {
      const { targets } = lineplot.reduceAll(base(), [
        lineplot.setLineColor({ key: "l1", color: color.construct("#0000ff") }),
      ]);
      expect(targets).toEqual(["line:l1"]);
    });
  });

  describe("setLine (full object)", () => {
    it("should replace an existing line's full configuration and round-trip", () => {
      const state = createEmpty({ lines: [createLine("l1", "#ff0000")] });
      const next: lineplot.Line = {
        key: "l1",
        color: color.construct("#0000ff"),
        strokeWidth: 4,
        downsample: 2,
        downsampleMode: "average",
      };
      expect(apply(state, lineplot.setLine({ line: next })).lines[0]).toEqual(next);
      expect(roundTrip(state, lineplot.setLine({ line: next })).lines).toEqual(
        state.lines,
      );
    });
    it("should insert a new line and not record it as undoable", () => {
      const inserted: lineplot.Line = {
        key: "l1",
        strokeWidth: 2,
        downsample: 1,
        downsampleMode: "decimate",
      };
      const { targets } = lineplot.reduceAll(createEmpty(), [
        lineplot.setLine({ line: inserted }),
      ]);
      expect(targets).toEqual([]);
      expect(apply(createEmpty(), lineplot.setLine({ line: inserted })).lines).toEqual([
        inserted,
      ]);
    });
  });

  describe("setRule and removeRule", () => {
    it("should append the rule when its key is new", () => {
      const out = apply(
        createEmpty(),
        lineplot.setRule({ rule: createRule("r1", "max") }),
      );
      expect(out.rules).toEqual([createRule("r1", "max")]);
    });
    it("should replace the rule in place when its key already exists", () => {
      const state = createEmpty({ rules: [createRule("r1", "max")] });
      const out = apply(state, lineplot.setRule({ rule: createRule("r1", "ceiling") }));
      expect(out.rules).toEqual([createRule("r1", "ceiling")]);
    });
    it("should drop the rule on removeRule", () => {
      const state = createEmpty({
        rules: [createRule("r1", "a"), createRule("r2", "b")],
      });
      const out = apply(state, lineplot.removeRule({ key: "r1" }));
      expect(out.rules).toEqual([createRule("r2", "b")]);
    });
    it("should round-trip add via the captured inverse removeRule", () => {
      const state = createEmpty();
      expect(
        roundTrip(state, lineplot.setRule({ rule: createRule("r1", "max") })).rules,
      ).toEqual([]);
    });
    it("should round-trip remove via the captured inverse setRule", () => {
      const state = createEmpty({ rules: [createRule("r1", "max")] });
      expect(roundTrip(state, lineplot.removeRule({ key: "r1" })).rules).toEqual(
        state.rules,
      );
    });
    it("should target the rule on both setRule and removeRule", () => {
      const state = createEmpty({ rules: [createRule("r1", "max")] });
      expect(
        lineplot.reduceAll(createEmpty(), [
          lineplot.setRule({ rule: createRule("r1", "max") }),
        ]).targets,
      ).toEqual(["rule:r1"]);
      expect(
        lineplot.reduceAll(state, [lineplot.removeRule({ key: "r1" })]).targets,
      ).toEqual(["rule:r1"]);
    });
  });

  describe("fine-grained rule actions", () => {
    const base = (): lineplot.LinePlot =>
      createEmpty({ rules: [createRule("r1", "max")] });
    it.each([
      {
        label: "setRuleLabel",
        action: lineplot.setRuleLabel({ key: "r1", label: "min" }),
        expected: { label: "min" },
      },
      {
        label: "setRuleColor",
        action: lineplot.setRuleColor({ key: "r1", color: color.construct("#00ff00") }),
        expected: { color: color.construct("#00ff00") },
      },
      {
        label: "setRuleAxis",
        action: lineplot.setRuleAxis({ key: "r1", axis: "y2" }),
        expected: { axis: "y2" },
      },
      {
        label: "setRuleLineWidth",
        action: lineplot.setRuleLineWidth({ key: "r1", lineWidth: 3 }),
        expected: { lineWidth: 3 },
      },
      {
        label: "setRuleLineDash",
        action: lineplot.setRuleLineDash({ key: "r1", lineDash: 4 }),
        expected: { lineDash: 4 },
      },
      {
        label: "setRuleUnits",
        action: lineplot.setRuleUnits({ key: "r1", units: "psi" }),
        expected: { units: "psi" },
      },
      {
        label: "setRulePosition",
        action: lineplot.setRulePosition({ key: "r1", position: 42 }),
        expected: { position: 42 },
      },
    ])("$label sets its field and round-trips via inverse", ({ action, expected }) => {
      const state = base();
      expect(apply(state, action).rules[0]).toMatchObject(expected);
      expect(roundTrip(state, action).rules[0]).toEqual(state.rules[0]);
    });

    it("is a no-op when the rule does not exist", () => {
      const { targets } = lineplot.reduceAll(createEmpty(), [
        lineplot.setRuleLabel({ key: "missing", label: "x" }),
      ]);
      expect(targets).toEqual([]);
    });

    it("targets the rule so distinct rules are independent", () => {
      const { targets } = lineplot.reduceAll(base(), [
        lineplot.setRuleLabel({ key: "r1", label: "x" }),
      ]);
      expect(targets).toEqual(["rule:r1"]);
    });
  });

  describe("lineplot.reduceAll", () => {
    it("should apply actions sequentially", () => {
      const out = apply(
        createEmpty({ name: "old" }),
        lineplot.rename({ name: "renamed" }),
        lineplot.addChannel({ axisKey: "y1", channel: 1 }),
        lineplot.addChannel({ axisKey: "y1", channel: 2 }),
      );
      expect(out.name).toEqual("renamed");
      expect(out.channels.y1).toEqual([1, 2]);
    });
    it("should collect inverses in reverse application order so undo replays them correctly", () => {
      const { inverse } = lineplot.reduceAll(createEmpty(), [
        lineplot.rename({ name: "a" }),
        lineplot.rename({ name: "b" }),
      ]);
      // first inverse popped is for the LAST action applied
      expect(inverse[0]).toEqual(lineplot.rename({ name: "a" }));
      expect(inverse[1]).toEqual(lineplot.rename({ name: "" }));
    });
    it("should deduplicate targets", () => {
      const created = createEmpty();
      const { targets } = lineplot.reduceAll(created, [
        lineplot.rename({ name: "a" }),
        lineplot.rename({ name: "b" }),
      ]);
      expect(targets).toEqual([created.key]);
    });
    it("should union the distinct per-resource targets it touched", () => {
      const { targets } = lineplot.reduceAll(createEmpty(), [
        lineplot.addChannel({ axisKey: "y1", channel: 1 }),
        lineplot.addChannel({ axisKey: "y2", channel: 2 }),
        lineplot.setRule({ rule: createRule("r1", "max") }),
      ]);
      expect(new Set(targets)).toEqual(new Set(["channel:1", "channel:2", "rule:r1"]));
    });
  });
});

describe("actionZ wire format", () => {
  it("should reject an unknown action type", () => {
    expect(() => lineplot.actionZ.parse({ type: "bogus", bogus: {} })).toThrow();
  });
  it("should accept a known action with its payload field", () => {
    const parsed = lineplot.actionZ.parse({ type: "rename", rename: { name: "x" } });
    expect(parsed.type).toEqual("rename");
  });
});

describe("scopedActionZ", () => {
  it("should parse a representative scoped action emitted by the server", () => {
    const parsed = lineplot.scopedActionZ.parse({
      key: "00000000-0000-4000-8000-000000000001",
      dispatchKey: "client-xyz",
      seq: 7,
      actions: [{ type: "rename", rename: { name: "new" } }],
    });
    expect(parsed.dispatchKey).toEqual("client-xyz");
    expect(parsed.seq).toEqual(7);
    expect(parsed.actions).toHaveLength(1);
  });
  it("should default seq to 0 when absent", () => {
    const parsed = lineplot.scopedActionZ.parse({
      key: "00000000-0000-4000-8000-000000000001",
      dispatchKey: "client",
      actions: [],
    });
    expect(parsed.seq).toEqual(0);
  });
});
