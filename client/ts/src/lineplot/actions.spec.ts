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

import { reduceAll } from "@/lineplot/actions";
import {
  type Action,
  actionZ,
  addChannel,
  addRange,
  removeChannel,
  removeRange,
  removeRule,
  rename,
  scopedActionZ,
  setAxis,
  setLegend,
  setLine,
  setRule,
  setTitle,
  setXChannel,
} from "@/lineplot/actions.gen";
import { type Axis, type Line, type LinePlot, type Rule } from "@/lineplot/types.gen";

const zeroAxis = (key: Axis["key"]): Axis => ({
  key,
  label: "",
  labelDirection: key.startsWith("y") ? "y" : "x",
  labelLevel: "small",
  bounds: { lower: 0, upper: 0 },
  autoBounds: { lower: true, upper: true },
  tickSpacing: 75,
  ...(key.startsWith("x") ? ({ type: "time" } as const) : {}),
});

const empty = (overrides: Partial<LinePlot> = {}): LinePlot => ({
  key: "00000000-0000-0000-0000-000000000000",
  name: "",
  title: { level: "h4", visible: false },
  legend: {
    visible: true,
    position: {
      x: 50,
      y: 50,
      root: { x: "left", y: "top" },
      units: { x: "px", y: "px" },
    },
  },
  channels: { x1: 0, x2: 0, y1: [], y2: [], y3: [], y4: [] },
  ranges: { x1: [], x2: [] },
  axes: {
    x1: zeroAxis("x1"),
    x2: zeroAxis("x2"),
    y1: zeroAxis("y1"),
    y2: zeroAxis("y2"),
    y3: zeroAxis("y3"),
    y4: zeroAxis("y4"),
  },
  lines: [],
  rules: [],
  ...overrides,
});

const apply = (state: LinePlot, ...actions: Action[]): LinePlot =>
  reduceAll(state, actions).next;

// roundTrip applies actions then applies their captured inverses in reverse.
// The result must equal the input state for actions that fully invert.
const roundTrip = (state: LinePlot, ...actions: Action[]): LinePlot => {
  const { next, inverse } = reduceAll(state, actions);
  return reduceAll(next, inverse).next;
};

const line = (key: string, hex: string): Line => ({
  key,
  color: color.construct(hex),
  strokeWidth: 1,
  downsample: 1,
  downsampleMode: "decimate",
});

const rule = (key: string, label: string): Rule => ({
  key,
  label,
  color: color.construct("#888888"),
  axis: "y1",
  lineWidth: 1,
  lineDash: 0,
  units: "",
  position: 0,
});

describe("lineplot reducer", () => {
  describe("rename", () => {
    it("should set the line plot name", () => {
      const out = apply(empty({ name: "old" }), rename({ name: "new" }));
      expect(out.name).toEqual("new");
    });
    it("should round-trip the previous name through its inverse", () => {
      const state = empty({ name: "before" });
      expect(roundTrip(state, rename({ name: "after" })).name).toEqual("before");
    });
    it("should target the plot key so plot-level edits invalidate each other", () => {
      const state = empty({ key: "11111111-1111-1111-1111-111111111111" });
      const { targets } = reduceAll(state, [rename({ name: "x" })]);
      expect(targets).toEqual([state.key]);
    });
  });

  describe("setTitle", () => {
    it("should replace the title configuration", () => {
      const out = apply(empty(), setTitle({ title: { level: "h2", visible: true } }));
      expect(out.title).toEqual({ level: "h2", visible: true });
    });
    it("should round-trip the previous title through its inverse", () => {
      const state = empty({ title: { level: "h4", visible: false } });
      expect(
        roundTrip(state, setTitle({ title: { level: "h1", visible: true } })).title,
      ).toEqual(state.title);
    });
  });

  describe("setLegend", () => {
    it("should replace the legend configuration", () => {
      const out = apply(
        empty(),
        setLegend({
          legend: { visible: false, position: { x: 10, y: 20 } },
        }),
      );
      expect(out.legend.visible).toEqual(false);
      expect(out.legend.position.x).toEqual(10);
    });
    it("should round-trip the previous legend through its inverse", () => {
      const state = empty();
      const swapped = apply(
        state,
        setLegend({ legend: { visible: false, position: { x: 1, y: 2 } } }),
      );
      expect(swapped.legend).not.toEqual(state.legend);
      expect(
        roundTrip(
          state,
          setLegend({
            legend: { visible: false, position: { x: 1, y: 2 } },
          }),
        ).legend,
      ).toEqual(state.legend);
    });
  });

  describe("addChannel", () => {
    it("should append the channel to the named y-axis", () => {
      const out = apply(empty(), addChannel({ axisKey: "y1", channel: 42 }));
      expect(out.channels.y1).toEqual([42]);
    });
    it("should be a no-op when the channel is already present", () => {
      const state = empty({
        channels: { x1: 0, x2: 0, y1: [42], y2: [], y3: [], y4: [] },
      });
      const out = apply(state, addChannel({ axisKey: "y1", channel: 42 }));
      expect(out.channels.y1).toEqual([42]);
    });
    it("should round-trip through removeChannel as its inverse", () => {
      const state = empty();
      expect(
        roundTrip(state, addChannel({ axisKey: "y2", channel: 7 })).channels.y2,
      ).toEqual([]);
    });
    it("should target the channel so distinct channels do not invalidate each other", () => {
      const { targets } = reduceAll(empty(), [
        addChannel({ axisKey: "y1", channel: 42 }),
      ]);
      expect(targets).toEqual(["channel:42"]);
    });
  });

  describe("removeChannel", () => {
    it("should drop the channel from the named y-axis", () => {
      const state = empty({
        channels: { x1: 0, x2: 0, y1: [1, 2, 3], y2: [], y3: [], y4: [] },
      });
      const out = apply(state, removeChannel({ axisKey: "y1", channel: 2 }));
      expect(out.channels.y1).toEqual([1, 3]);
    });
    it("should be a no-op when the channel is not present", () => {
      const state = empty({
        channels: { x1: 0, x2: 0, y1: [1], y2: [], y3: [], y4: [] },
      });
      const out = apply(state, removeChannel({ axisKey: "y1", channel: 99 }));
      expect(out.channels.y1).toEqual([1]);
    });
    it("should round-trip through addChannel as its inverse", () => {
      const state = empty({
        channels: { x1: 0, x2: 0, y1: [5], y2: [], y3: [], y4: [] },
      });
      expect(
        roundTrip(state, removeChannel({ axisKey: "y1", channel: 5 })).channels.y1,
      ).toEqual([5]);
    });
  });

  describe("setXChannel", () => {
    it("should set the channel on the named x-axis", () => {
      const out = apply(empty(), setXChannel({ axisKey: "x1", channel: 99 }));
      expect(out.channels.x1).toEqual(99);
    });
    it("should round-trip the previous x-channel through its inverse", () => {
      const state = empty({
        channels: { x1: 42, x2: 0, y1: [], y2: [], y3: [], y4: [] },
      });
      expect(
        roundTrip(state, setXChannel({ axisKey: "x1", channel: 99 })).channels.x1,
      ).toEqual(42);
    });
    it("should target the x-axis so x1 and x2 edits are independent", () => {
      const { targets } = reduceAll(empty(), [
        setXChannel({ axisKey: "x2", channel: 99 }),
      ]);
      expect(targets).toEqual(["axis:x2"]);
    });
  });

  describe("addRange and removeRange", () => {
    const r1 = "00000000-0000-0000-0000-000000000001";
    const r2 = "00000000-0000-0000-0000-000000000002";

    it("should append the range to the named x-axis", () => {
      const out = apply(empty(), addRange({ axisKey: "x1", range: r1 }));
      expect(out.ranges.x1).toEqual([r1]);
    });
    it("should drop the range from the named x-axis", () => {
      const state = empty({ ranges: { x1: [r1, r2], x2: [] } });
      const out = apply(state, removeRange({ axisKey: "x1", range: r1 }));
      expect(out.ranges.x1).toEqual([r2]);
    });
    it("should round-trip add then remove via the captured inverses", () => {
      const state = empty();
      expect(
        roundTrip(state, addRange({ axisKey: "x2", range: r1 })).ranges.x2,
      ).toEqual([]);
    });
    it("should target the range so distinct ranges do not invalidate each other", () => {
      const { targets } = reduceAll(empty(), [addRange({ axisKey: "x1", range: r1 })]);
      expect(targets).toEqual([`range:${r1}`]);
    });
  });

  describe("setAxis", () => {
    it("should replace the axis configuration at the target key", () => {
      const next: Axis = { ...zeroAxis("y1"), label: "pressure", tickSpacing: 50 };
      const out = apply(empty(), setAxis({ axis: next }));
      expect(out.axes.y1.label).toEqual("pressure");
      expect(out.axes.y1.tickSpacing).toEqual(50);
    });
    it("should round-trip the previous axis through its inverse", () => {
      const state = empty();
      const swapped = { ...zeroAxis("x1"), label: "time" };
      expect(roundTrip(state, setAxis({ axis: swapped })).axes.x1).toEqual(
        state.axes.x1,
      );
    });
    it("should target the edited axis so distinct axes are independent", () => {
      const { targets } = reduceAll(empty(), [setAxis({ axis: zeroAxis("y3") })]);
      expect(targets).toEqual(["axis:y3"]);
    });
  });

  describe("setLine", () => {
    it("should append the line when its key is new", () => {
      const out = apply(empty(), setLine({ line: line("l1", "#ff0000") }));
      expect(out.lines).toEqual([line("l1", "#ff0000")]);
    });
    it("should replace the line in place when its key already exists", () => {
      const state = empty({ lines: [line("l1", "#ff0000"), line("l2", "#00ff00")] });
      const out = apply(state, setLine({ line: line("l1", "#0000ff") }));
      expect(out.lines).toEqual([line("l1", "#0000ff"), line("l2", "#00ff00")]);
    });
    it("should round-trip the previous line through its inverse on update", () => {
      const state = empty({ lines: [line("l1", "#ff0000")] });
      expect(roundTrip(state, setLine({ line: line("l1", "#0000ff") })).lines).toEqual(
        state.lines,
      );
    });
    it("should report no target for a fresh line insert so it is not recorded as undoable", () => {
      const { targets } = reduceAll(empty(), [
        setLine({ line: line("l1", "#ff0000") }),
      ]);
      expect(targets).toEqual([]);
    });
    it("should target the line on update so distinct lines are independent", () => {
      const state = empty({ lines: [line("l1", "#ff0000")] });
      const { targets } = reduceAll(state, [setLine({ line: line("l1", "#0000ff") })]);
      expect(targets).toEqual(["line:l1"]);
    });
  });

  describe("setRule and removeRule", () => {
    it("should append the rule when its key is new", () => {
      const out = apply(empty(), setRule({ rule: rule("r1", "max") }));
      expect(out.rules).toEqual([rule("r1", "max")]);
    });
    it("should replace the rule in place when its key already exists", () => {
      const state = empty({ rules: [rule("r1", "max")] });
      const out = apply(state, setRule({ rule: rule("r1", "ceiling") }));
      expect(out.rules).toEqual([rule("r1", "ceiling")]);
    });
    it("should drop the rule on removeRule", () => {
      const state = empty({ rules: [rule("r1", "a"), rule("r2", "b")] });
      const out = apply(state, removeRule({ key: "r1" }));
      expect(out.rules).toEqual([rule("r2", "b")]);
    });
    it("should round-trip add via the captured inverse removeRule", () => {
      const state = empty();
      expect(roundTrip(state, setRule({ rule: rule("r1", "max") })).rules).toEqual([]);
    });
    it("should round-trip remove via the captured inverse setRule", () => {
      const state = empty({ rules: [rule("r1", "max")] });
      expect(roundTrip(state, removeRule({ key: "r1" })).rules).toEqual(state.rules);
    });
    it("should target the rule on both setRule and removeRule", () => {
      const state = empty({ rules: [rule("r1", "max")] });
      expect(
        reduceAll(empty(), [setRule({ rule: rule("r1", "max") })]).targets,
      ).toEqual(["rule:r1"]);
      expect(reduceAll(state, [removeRule({ key: "r1" })]).targets).toEqual([
        "rule:r1",
      ]);
    });
  });

  describe("reduceAll", () => {
    it("should apply actions sequentially", () => {
      const out = apply(
        empty({ name: "old" }),
        rename({ name: "renamed" }),
        addChannel({ axisKey: "y1", channel: 1 }),
        addChannel({ axisKey: "y1", channel: 2 }),
      );
      expect(out.name).toEqual("renamed");
      expect(out.channels.y1).toEqual([1, 2]);
    });
    it("should collect inverses in reverse application order so undo replays them correctly", () => {
      const { inverse } = reduceAll(empty(), [
        rename({ name: "a" }),
        rename({ name: "b" }),
      ]);
      // first inverse popped is for the LAST action applied
      expect(inverse[0]).toEqual(rename({ name: "a" }));
      expect(inverse[1]).toEqual(rename({ name: "" }));
    });
    it("should deduplicate targets", () => {
      const { targets } = reduceAll(empty(), [
        rename({ name: "a" }),
        rename({ name: "b" }),
      ]);
      expect(targets).toEqual(["00000000-0000-0000-0000-000000000000"]);
    });
    it("should union the distinct per-resource targets it touched", () => {
      const { targets } = reduceAll(empty(), [
        addChannel({ axisKey: "y1", channel: 1 }),
        addChannel({ axisKey: "y2", channel: 2 }),
        setRule({ rule: rule("r1", "max") }),
      ]);
      expect(new Set(targets)).toEqual(new Set(["channel:1", "channel:2", "rule:r1"]));
    });
  });
});

describe("actionZ wire format", () => {
  it("should reject an unknown action type", () => {
    expect(() => actionZ.parse({ type: "bogus", bogus: {} })).toThrow();
  });
  it("should accept a known action with its payload field", () => {
    const parsed = actionZ.parse({ type: "rename", rename: { name: "x" } });
    expect(parsed.type).toEqual("rename");
  });
});

describe("scopedActionZ", () => {
  it("should parse a representative scoped action emitted by the server", () => {
    const parsed = scopedActionZ.parse({
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
    const parsed = scopedActionZ.parse({
      key: "00000000-0000-4000-8000-000000000001",
      dispatchKey: "client",
      actions: [],
    });
    expect(parsed.seq).toEqual(0);
  });
});
