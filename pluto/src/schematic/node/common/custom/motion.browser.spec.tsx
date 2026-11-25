// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import type { ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import { commands } from "vitest/browser";
import { render } from "vitest-browser-react";

import { Custom } from "@/schematic/node/common/custom";
import { Primitive } from "@/schematic/node/common/primitive";

const MARKUP =
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
  '<rect class="rotor" width="4" height="4" transform="translate(1 1)"/>' +
  '<path class="pipe" d="M0 5H10" stroke="black" stroke-dasharray="4 2" ' +
  'stroke-dashoffset="1"/>' +
  '<rect class="liquid" width="10" height="10"/>' +
  '<g class="arm" transform="scale(2)">' +
  '<rect class="slider" width="1" height="1"/></g>' +
  "</svg>";

const PARTS: schematic.symbol.Part[] = [
  { key: "rotor", name: "Rotor", selectors: [".rotor"] },
  { key: "pipe", name: "Pipe", selectors: [".pipe"] },
  { key: "liquid", name: "Liquid", selectors: [".liquid"] },
  { key: "slider", name: "Slider", selectors: [".slider"] },
];

const STATES: schematic.symbol.State[] = [
  { type: "number", key: "speed", name: "Speed", bounds: { lower: 0, upper: 100 } },
  {
    type: "enumeration",
    key: "active",
    name: "Active",
    match: { type: "exact" },
    default: 0,
  },
];

const createSpec = (
  looks: schematic.symbol.Look[],
  parts: schematic.symbol.Part[] = PARTS,
): schematic.symbol.Spec => ({
  drawing: { type: "svg", markup: MARKUP },
  parts,
  handles: [],
  states: STATES,
  commands: [],
  looks,
  interactions: [],
  variant: "static",
  scale: 1,
  strokeScaled: false,
});

const SPIN: schematic.symbol.Look = {
  key: "spin",
  part: "rotor",
  motion: { type: "spin", rate: 0.5 },
};

// stubFrames fakes requestAnimationFrame, so a frame runs only when a spec calls frame.
const stubFrames = () => {
  const callbacks = new Map<number, FrameRequestCallback>();
  let next = 0;
  vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
    callbacks.set(++next, cb);
    return next;
  });
  vi.stubGlobal("cancelAnimationFrame", (handle: number) => {
    callbacks.delete(handle);
  });
  const frame = (now: number): void => {
    const due = [...callbacks.values()];
    callbacks.clear();
    for (const cb of due) cb(now);
  };
  return { frame, pending: () => callbacks.size };
};

const Harness = ({ spec, ...rest }: Partial<Custom.DrawingProps>): ReactElement => (
  <Custom.MotionProvider>
    <Primitive.Div>
      <Custom.Drawing spec={spec ?? createSpec([SPIN])} {...rest} />
    </Primitive.Div>
  </Custom.MotionProvider>
);

const setup = async (props: Partial<Custom.DrawingProps> = {}) => {
  const { frame, pending } = stubFrames();
  const result = await render(<Harness {...props} />);
  const attr = (selector: string, name: string): string | null =>
    result.container.querySelector(selector)?.getAttribute(name) ?? null;
  const rerender = (next: Partial<Custom.DrawingProps>) =>
    result.rerender(<Harness {...props} {...next} />);
  return { ...result, frame, pending, attr, rerender };
};

const reading = (value: number, stale = false): Custom.Reading => ({ value, stale });

describe("Custom.Drawing motion", () => {
  afterEach(async () => {
    vi.unstubAllGlobals();
    await commands.emulateReducedMotion(false);
  });

  describe("provider", () => {
    it("should throw for a spec whose looks move outside a MotionProvider", () => {
      expect(() =>
        renderToStaticMarkup(<Custom.Drawing spec={createSpec([SPIN])} />),
      ).toThrow("Custom.Drawing draws motion only inside a Custom.MotionProvider");
    });

    it("should draw a still spec outside a MotionProvider", async () => {
      const { container } = await render(
        <Primitive.Div>
          <Custom.Drawing spec={createSpec([])} />
        </Primitive.Div>,
      );
      expect(container.querySelector(".rotor")).not.toBeNull();
    });
  });

  describe("loop", () => {
    it("should ask for one frame for two moving drawings", async () => {
      const { pending } = stubFrames();
      await render(
        <Custom.MotionProvider>
          <Primitive.Div>
            <Custom.Drawing spec={createSpec([SPIN])} />
          </Primitive.Div>
          <Primitive.Div>
            <Custom.Drawing spec={createSpec([SPIN])} />
          </Primitive.Div>
        </Custom.MotionProvider>,
      );
      expect(pending()).toBe(1);
    });
  });

  describe("spin", () => {
    it("should turn a part around the center of its own box at its rate", async () => {
      const { pending, frame, attr } = await setup();
      expect(pending()).toBe(1);
      frame(0);
      frame(500);
      expect(attr(".rotor", "transform")).toBe("rotate(90 3 3) translate(1 1)");
    });

    it("should turn a part in a scaled group around its own center", async () => {
      const look = { ...SPIN, part: "slider" };
      const { frame, attr } = await setup({ spec: createSpec([look]) });
      frame(0);
      frame(500);
      expect(attr(".slider", "transform")).toBe("rotate(90 0.5 0.5)");
    });

    it("should turn a part around its pivot", async () => {
      const parts = PARTS.map((p) =>
        p.key === "rotor" ? { ...p, pivot: { x: 0.2, y: 0.4 } } : p,
      );
      const { frame, attr } = await setup({ spec: createSpec([SPIN], parts) });
      frame(0);
      frame(500);
      expect(attr(".rotor", "transform")).toBe("rotate(90 2 4) translate(1 1)");
    });

    it("should turn the whole drawing around the center of its box", async () => {
      const look: schematic.symbol.Look = {
        key: "spin",
        motion: { type: "spin", rate: 0.5 },
      };
      const { frame, container } = await setup({ spec: createSpec([look]) });
      frame(0);
      frame(500);
      const drawing = container.querySelector(".rotor")?.parentElement;
      expect(drawing?.getAttribute("transform")).toBe("rotate(90 5 5)");
    });

    it("should turn a part in a turning group around its own center", async () => {
      const parts: schematic.symbol.Part[] = [
        { key: "arm", name: "Arm", selectors: [".arm"], pivot: { x: 0, y: 0 } },
        { key: "slider", name: "Slider", selectors: [".slider"] },
      ];
      const looks: schematic.symbol.Look[] = [
        { key: "arm", part: "arm", motion: { type: "spin", rate: 0.5 } },
        {
          key: "slider",
          part: "slider",
          motion: { type: "spin", rate: 1, state: "speed" },
        },
      ];
      const { frame, attr, rerender } = await setup({
        spec: createSpec(looks, parts),
        readings: { speed: reading(0) },
      });
      frame(0);
      frame(500);
      await rerender({ readings: { speed: reading(100) } });
      frame(1000);
      frame(1250);
      expect(attr(".slider", "transform")).toBe("rotate(90 0.5 0.5)");
    });

    it("should measure the center again when a hidden drawing shows", async () => {
      const { frame, attr, container } = await setup();
      container.style.display = "none";
      frame(0);
      container.style.display = "";
      frame(500);
      expect(attr(".rotor", "transform")).toBe("rotate(90 3 3) translate(1 1)");
    });

    it("should turn a part around its new pivot when the pivot changes", async () => {
      const { frame, attr, rerender } = await setup();
      frame(0);
      frame(500);
      const parts = PARTS.map((p) =>
        p.key === "rotor" ? { ...p, pivot: { x: 0.2, y: 0.4 } } : p,
      );
      await rerender({ spec: createSpec([SPIN], parts) });
      frame(1000);
      frame(1500);
      expect(attr(".rotor", "transform")).toBe("rotate(90 2 4) translate(1 1)");
    });

    it("should draw the part as drawn when its motion changes kind", async () => {
      const flow: schematic.symbol.Look = {
        key: "flow",
        part: "rotor",
        motion: { type: "flow", rate: 0.5 },
      };
      const { pending, frame, attr, rerender } = await setup();
      frame(0);
      frame(500);
      await rerender({ spec: createSpec([flow]) });
      expect(attr(".rotor", "transform")).toBe("translate(1 1)");
      expect(pending()).toBe(1);
    });

    it("should read bounds that the Core refuses as fraction 0", async () => {
      const look = {
        ...SPIN,
        motion: { type: "spin" as const, rate: 1, state: "speed" },
      };
      const valid = createSpec([look]);
      const states = [{ ...STATES[0], bounds: { lower: 50, upper: 50 } }, STATES[1]];
      const { pending, frame, attr, rerender } = await setup({
        spec: { ...valid, states },
        readings: { speed: reading(50) },
      });
      expect(pending()).toBe(0);
      await rerender({ spec: valid });
      frame(0);
      frame(1000);
      expect(attr(".rotor", "transform")).toBe("rotate(180 3 3) translate(1 1)");
    });

    it("should turn the other way at a negative rate", async () => {
      const look = { ...SPIN, motion: { type: "spin" as const, rate: -0.5 } };
      const { frame, attr } = await setup({ spec: createSpec([look]) });
      frame(0);
      frame(500);
      expect(attr(".rotor", "transform")).toBe("rotate(270 3 3) translate(1 1)");
    });

    it("should turn at its rate times the fraction of its state", async () => {
      const look = {
        ...SPIN,
        motion: { type: "spin" as const, rate: 1, state: "speed" },
      };
      const { frame, attr } = await setup({
        spec: createSpec([look]),
        readings: { speed: reading(25) },
      });
      frame(0);
      frame(1000);
      expect(attr(".rotor", "transform")).toBe("rotate(90 3 3) translate(1 1)");
    });

    it("should leave the loop while its state reads 0, and join it again", async () => {
      const look = {
        ...SPIN,
        motion: { type: "spin" as const, rate: 1, state: "speed" },
      };
      const spec = createSpec([look]);
      const { pending, rerender } = await setup({
        spec,
        readings: { speed: reading(0) },
      });
      expect(pending()).toBe(0);
      await rerender({ readings: { speed: reading(50) } });
      expect(pending()).toBe(1);
    });

    it("should stop and keep its angle while its state is stale", async () => {
      const look = {
        ...SPIN,
        motion: { type: "spin" as const, rate: 1, state: "speed" },
      };
      const spec = createSpec([look]);
      const { pending, frame, attr, rerender } = await setup({
        spec,
        readings: { speed: reading(100) },
      });
      frame(0);
      frame(250);
      await rerender({ readings: { speed: reading(100, true) } });
      expect(pending()).toBe(0);
      expect(attr(".rotor", "transform")).toBe("rotate(90 3 3) translate(1 1)");
    });

    it("should stop while its state is unknown", async () => {
      const look = {
        ...SPIN,
        motion: { type: "spin" as const, rate: 1, state: "speed" },
      };
      const { pending } = await setup({
        spec: createSpec([look]),
        readings: { speed: reading(NaN) },
      });
      expect(pending()).toBe(0);
    });

    it("should stop while a state that its look reads in when is stale", async () => {
      const look = {
        ...SPIN,
        when: { type: "equals" as const, state: "active", values: [1] },
      };
      const spec = createSpec([look]);
      const { pending, rerender } = await setup({
        spec,
        readings: { active: reading(1) },
      });
      expect(pending()).toBe(1);
      await rerender({ readings: { active: reading(1, true) } });
      expect(pending()).toBe(0);
    });

    it("should leave the loop and draw as drawn when its look stops", async () => {
      const look = {
        ...SPIN,
        when: { type: "equals" as const, state: "active", values: [1] },
      };
      const spec = createSpec([look]);
      const { pending, frame, attr, rerender } = await setup({
        spec,
        readings: { active: reading(1) },
      });
      frame(0);
      frame(500);
      await rerender({ readings: { active: reading(0) } });
      expect(pending()).toBe(0);
      expect(attr(".rotor", "transform")).toBe("translate(1 1)");
    });

    it("should leave the loop while a later look hides the part", async () => {
      const hide: schematic.symbol.Look = { key: "hide", part: "rotor", hidden: true };
      const { pending } = await setup({ spec: createSpec([SPIN, hide]) });
      expect(pending()).toBe(0);
    });

    it("should read an unknown number state in when at its lower bound", async () => {
      const hide: schematic.symbol.Look = {
        key: "hide",
        part: "rotor",
        when: { type: "equals", state: "speed", values: [0] },
        hidden: true,
      };
      const { pending, rerender } = await setup({
        spec: createSpec([SPIN, hide]),
        readings: { speed: reading(50) },
      });
      expect(pending()).toBe(1);
      await rerender({ readings: { speed: reading(NaN) } });
      expect(pending()).toBe(0);
    });

    it("should stop on a node with motion disabled", async () => {
      const { pending, rerender } = await setup();
      await rerender({ motionDisabled: true });
      expect(pending()).toBe(0);
    });

    it("should stop under the Reduce motion setting", async () => {
      const { pending } = await setup();
      expect(pending()).toBe(1);
      await commands.emulateReducedMotion(true);
      await expect.poll(() => pending()).toBe(0);
    });

    it("should leave the loop when the drawing is removed", async () => {
      const { pending, unmount } = await setup();
      await unmount();
      expect(pending()).toBe(0);
    });

    it("should move the new drawing when the markup changes", async () => {
      const spec = createSpec([SPIN]);
      const { pending, frame, attr, rerender } = await setup({ spec });
      await rerender({
        spec: { ...spec, drawing: { type: "svg", markup: MARKUP.replace("4", "3") } },
      });
      expect(pending()).toBe(1);
      frame(0);
      frame(500);
      expect(attr(".rotor", "transform")).toBe("rotate(90 2.5 3) translate(1 1)");
    });
  });

  describe("flow", () => {
    it("should move the dashes one pattern a cycle from their offset", async () => {
      const look: schematic.symbol.Look = {
        key: "flow",
        part: "pipe",
        motion: { type: "flow", rate: 0.5 },
      };
      const { frame, attr } = await setup({ spec: createSpec([look]) });
      frame(0);
      frame(1000);
      expect(attr(".pipe", "stroke-dashoffset")).toBe("-2");
    });
  });

  describe("blink", () => {
    const blink = (period: number): schematic.symbol.Look => ({
      key: "blink",
      part: "rotor",
      blink: { period },
    });

    it("should dim the part for the second half of each period", async () => {
      const { frame, attr } = await setup({ spec: createSpec([blink(1e9)]) });
      frame(0);
      expect(attr(".rotor", "opacity")).toBe("1");
      frame(600);
      expect(attr(".rotor", "opacity")).toBe("0.2");
    });

    it("should flash at most 2.5 times a second", async () => {
      const { frame, attr } = await setup({ spec: createSpec([blink(1e8)]) });
      frame(0);
      frame(150);
      expect(attr(".rotor", "opacity")).toBe("1");
      frame(250);
      expect(attr(".rotor", "opacity")).toBe("0.2");
    });

    it("should show the part again when it stops", async () => {
      const look = {
        ...blink(1e9),
        when: { type: "equals" as const, state: "active", values: [1] },
      };
      const spec = createSpec([look]);
      const { pending, frame, attr, rerender } = await setup({
        spec,
        readings: { active: reading(1) },
      });
      frame(0);
      frame(600);
      await rerender({ readings: { active: reading(1, true) } });
      expect(pending()).toBe(0);
      expect(attr(".rotor", "opacity")).toBe("1");
    });
  });

  describe("level", () => {
    const level = (side: schematic.symbol.LevelMotion["side"] = "bottom") =>
      createSpec([
        {
          key: "level",
          part: "liquid",
          motion: { type: "level", state: "speed", side },
        },
      ]);

    it("should fill a part from its side to the fraction of its state", async () => {
      const { pending, attr } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
      });
      expect(pending()).toBe(0);
      expect(attr(".liquid", "clip-path")).toBe("inset(75% 0 0 0)");
    });

    it("should fill from the top", async () => {
      const { attr } = await setup({
        spec: level("top"),
        readings: { speed: reading(25) },
      });
      expect(attr(".liquid", "clip-path")).toBe("inset(0 0 75% 0)");
    });

    it("should join the loop while it moves to a new fraction", async () => {
      const { pending, frame, attr, rerender } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
      });
      await rerender({ readings: { speed: reading(75) } });
      expect(pending()).toBe(1);
      frame(0);
      frame(100);
      expect(attr(".liquid", "clip-path")).not.toBe("inset(25% 0 0 0)");
      frame(200);
      expect(attr(".liquid", "clip-path")).toBe("inset(25% 0 0 0)");
      expect(pending()).toBe(0);
    });

    it("should slide on the curve of the modal animation", async () => {
      const { frame, attr, rerender } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
      });
      await rerender({ readings: { speed: reading(75) } });
      frame(0);
      frame(75);
      expect(attr(".liquid", "clip-path")).toBe("inset(50% 0 0 0)");
    });

    it("should land a value that arrives mid-slide when the slide ends", async () => {
      const { frame, attr, rerender } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
      });
      await rerender({ readings: { speed: reading(75) } });
      frame(0);
      frame(75);
      await rerender({ readings: { speed: reading(100) } });
      expect(attr(".liquid", "clip-path")).toBe("inset(50% 0 0 0)");
      frame(150);
      expect(attr(".liquid", "clip-path")).toBe("inset(0% 0 0 0)");
    });

    it("should jump to a new fraction on a node with motion disabled", async () => {
      const { pending, attr, rerender } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
        motionDisabled: true,
      });
      await rerender({ readings: { speed: reading(75) } });
      expect(pending()).toBe(0);
      expect(attr(".liquid", "clip-path")).toBe("inset(25% 0 0 0)");
    });

    it("should draw an empty level while its state is unknown", async () => {
      const { pending, attr, rerender } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
      });
      await rerender({ readings: { speed: reading(NaN) } });
      expect(pending()).toBe(0);
      expect(attr(".liquid", "clip-path")).toBe("inset(100% 0 0 0)");
    });

    it("should keep its last fraction while its state is stale", async () => {
      const { pending, attr, rerender } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
      });
      await rerender({ readings: { speed: reading(75, true) } });
      expect(pending()).toBe(0);
      expect(attr(".liquid", "clip-path")).toBe("inset(75% 0 0 0)");
    });

    it("should draw an edited look while its fraction holds", async () => {
      const { attr, rerender } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
      });
      await rerender({ spec: level("top") });
      expect(attr(".liquid", "clip-path")).toBe("inset(0 0 75% 0)");
    });
  });

  describe("rotate", () => {
    it("should turn a part to the angle at the fraction of its state", async () => {
      const look: schematic.symbol.Look = {
        key: "rotate",
        part: "rotor",
        motion: { type: "rotate", state: "speed", from: -45, to: 45 },
      };
      const { attr } = await setup({
        spec: createSpec([look]),
        readings: { speed: reading(75) },
      });
      expect(attr(".rotor", "transform")).toBe("rotate(22.5 3 3) translate(1 1)");
    });
  });

  describe("move", () => {
    it("should move a part by fractions of the box", async () => {
      const look: schematic.symbol.Look = {
        key: "move",
        part: "slider",
        motion: {
          type: "move",
          state: "speed",
          from: { x: 0, y: 0 },
          to: { x: 0.5, y: 0 },
        },
      };
      const { attr } = await setup({
        spec: createSpec([look]),
        readings: { speed: reading(100) },
      });
      expect(attr(".slider", "transform")).toBe("translate(2.5 0)");
    });
  });

  describe("kinds", () => {
    const MOVE: schematic.symbol.Look = {
      key: "move",
      part: "rotor",
      motion: {
        type: "move",
        state: "speed",
        from: { x: 0, y: 0 },
        to: { x: 0.5, y: 0 },
      },
    };

    it("should move a part, then spin it about its pivot", async () => {
      const { frame, attr } = await setup({
        spec: createSpec([SPIN, MOVE]),
        readings: { speed: reading(100) },
      });
      frame(0);
      frame(500);
      expect(attr(".rotor", "transform")).toBe(
        "translate(5 0) rotate(90 3 3) translate(1 1)",
      );
    });

    it("should add the angles of a rotate and a spin", async () => {
      const look: schematic.symbol.Look = {
        key: "rotate",
        part: "rotor",
        motion: { type: "rotate", state: "speed", from: 0, to: 90 },
      };
      const { frame, attr } = await setup({
        spec: createSpec([look, SPIN]),
        readings: { speed: reading(50) },
      });
      frame(0);
      frame(500);
      expect(attr(".rotor", "transform")).toBe("rotate(135 3 3) translate(1 1)");
    });

    it("should replace an earlier spin with a later spin", async () => {
      const fast = { ...SPIN, key: "fast", motion: { type: "spin" as const, rate: 1 } };
      const { frame, attr } = await setup({ spec: createSpec([SPIN, fast]) });
      frame(0);
      frame(500);
      expect(attr(".rotor", "transform")).toBe("rotate(180 3 3) translate(1 1)");
    });

    it("should fill and flow one part at once", async () => {
      const looks: schematic.symbol.Look[] = [
        {
          key: "level",
          part: "pipe",
          motion: { type: "level", state: "speed", side: "bottom" },
        },
        { key: "flow", part: "pipe", motion: { type: "flow", rate: 0.5 } },
      ];
      const { frame, attr } = await setup({
        spec: createSpec(looks),
        readings: { speed: reading(25) },
      });
      frame(0);
      frame(1000);
      expect(attr(".pipe", "clip-path")).toBe("inset(75% 0 0 0)");
      expect(attr(".pipe", "stroke-dashoffset")).toBe("-2");
    });

    it("should keep the move of a part when its spin stops", async () => {
      const spin = {
        ...SPIN,
        when: { type: "equals" as const, state: "active", values: [1] },
      };
      const { frame, attr, rerender } = await setup({
        spec: createSpec([MOVE, spin]),
        readings: { speed: reading(100), active: reading(1) },
      });
      frame(0);
      frame(500);
      await rerender({ readings: { speed: reading(100), active: reading(0) } });
      expect(attr(".rotor", "transform")).toBe("translate(5 0) translate(1 1)");
    });
  });

  describe("groups", () => {
    const arm = (selectors: string[]): schematic.symbol.Part[] => [
      ...PARTS,
      { key: "arm", name: "Arm", selectors },
    ];

    it("should turn a group about the center of its shapes at rest", async () => {
      const move: schematic.symbol.Look = {
        key: "move",
        part: "slider",
        motion: {
          type: "move",
          state: "speed",
          from: { x: 0, y: 0 },
          to: { x: 0.5, y: 0 },
        },
      };
      const { frame, attr } = await setup({
        spec: createSpec([move, { ...SPIN, part: "arm" }], arm([".arm"])),
        readings: { speed: reading(100) },
      });
      frame(0);
      frame(500);
      expect(attr(".arm", "transform")).toBe("rotate(90 1 1) scale(2)");
    });

    it("should turn an element in a selected group once, with the group", async () => {
      const { frame, attr } = await setup({
        spec: createSpec([{ ...SPIN, part: "arm" }], arm([".arm", ".slider"])),
      });
      frame(0);
      frame(500);
      expect(attr(".arm", "transform")).toBe("rotate(90 1 1) scale(2)");
      expect(attr(".slider", "transform")).toBeNull();
    });

    // setupBox spins a group with a nested svg from 2 to 6 and a rect from 6 to 8.
    const setupBox = (props: Partial<Custom.DrawingProps> = {}) => {
      const markup =
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><g class="box">' +
        '<svg x="2" y="2" width="4" height="4" viewBox="0 0 1 1">' +
        '<rect width="1" height="1"/></svg>' +
        '<rect class="far" x="6" y="6" width="2" height="2"/></g></svg>';
      const box = { key: "box", name: "Box", selectors: [".box"] };
      const spec = createSpec([{ ...SPIN, part: "box" }], [box]);
      return setup({ spec: { ...spec, drawing: { type: "svg", markup } }, ...props });
    };

    it("should turn a group about the center of a nested svg in it", async () => {
      const { frame, attr } = await setupBox();
      frame(0);
      frame(500);
      expect(attr(".box", "transform")).toBe("rotate(90 5 5)");
    });

    it("should turn a group about the center of a shape that a look hides", async () => {
      const { frame, attr } = await setupBox({
        paints: [{ selectors: [".far"], hidden: true }],
      });
      frame(0);
      frame(500);
      expect(attr(".box", "transform")).toBe("rotate(90 5 5)");
      expect(attr(".far", "style")).toBe("display: none;");
    });

    it("should measure a group with a nested svg again when its tab shows", async () => {
      const { frame, attr, container } = await setupBox();
      container.style.display = "none";
      frame(0);
      container.style.display = "";
      frame(500);
      expect(attr(".box", "transform")).toBe("rotate(90 5 5)");
    });

    it("should flow a dashed element in a selected group", async () => {
      const markup =
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><g class="line">' +
        '<path class="dash" d="M0 5H10" stroke="black" stroke-dasharray="4 2"/></g></svg>';
      const flow: schematic.symbol.Look = {
        key: "flow",
        part: "line",
        motion: { type: "flow", rate: 0.5 },
      };
      const line = { key: "line", name: "Line", selectors: [".line", ".dash"] };
      const spec = createSpec([flow], [line]);
      const { frame, attr } = await setup({
        spec: { ...spec, drawing: { type: "svg", markup } },
      });
      frame(0);
      frame(1000);
      expect(attr(".dash", "stroke-dashoffset")).toBe("-3");
    });
  });

  describe("shared elements", () => {
    const SHARED = [
      ...PARTS,
      { key: "rotor2", name: "Rotor 2", selectors: [".rotor"] },
    ];

    it("should compose the motions of two parts that select one element", async () => {
      const rotate: schematic.symbol.Look = {
        key: "rotate",
        part: "rotor",
        motion: { type: "rotate", state: "speed", from: 0, to: 90 },
      };
      const spin = { ...SPIN, part: "rotor2" };
      const { frame, attr, rerender } = await setup({
        spec: createSpec([rotate, spin], SHARED),
        readings: { speed: reading(50) },
      });
      frame(0);
      frame(500);
      await rerender({ readings: { speed: reading(100) } });
      frame(600);
      frame(750);
      expect(attr(".rotor", "transform")).toBe("rotate(225 3 3) translate(1 1)");
    });

    it("should draw the spin of the later look when two parts spin one element", async () => {
      // The blink puts the rotor part first in the loop, so the later look draws first.
      const looks: schematic.symbol.Look[] = [
        { key: "blink", part: "rotor", blink: { period: 1e9 } },
        { ...SPIN, key: "fast", part: "rotor2", motion: { type: "spin", rate: 1 } },
        SPIN,
      ];
      const { frame, attr } = await setup({ spec: createSpec(looks, SHARED) });
      frame(0);
      frame(500);
      expect(attr(".rotor", "transform")).toBe("rotate(90 3 3) translate(1 1)");
    });

    it("should hold an element still while the spin of the later look stops", async () => {
      const still = {
        ...SPIN,
        key: "still",
        part: "rotor2",
        motion: { type: "spin" as const, rate: 1, state: "speed" },
      };
      const { frame, attr } = await setup({
        spec: createSpec([SPIN, still], SHARED),
        readings: { speed: reading(0) },
      });
      frame(0);
      frame(500);
      expect(attr(".rotor", "transform")).toBe("translate(1 1)");
    });

    it("should draw the spin of the later look when the looks change order", async () => {
      const motion = { type: "spin" as const, rate: 1, state: "speed" };
      const slow = { ...SPIN, motion: { ...motion, rate: 0.5 } };
      const fast = { ...SPIN, key: "fast", part: "rotor2", motion };
      const { frame, attr, rerender } = await setup({
        spec: createSpec([slow, fast], SHARED),
        readings: { speed: reading(100) },
      });
      frame(0);
      frame(500);
      await rerender({ readings: { speed: reading(100, true) } });
      expect(attr(".rotor", "transform")).toBe("rotate(180 3 3) translate(1 1)");
      await rerender({
        spec: createSpec([fast, slow], SHARED),
        readings: { speed: reading(100, true) },
      });
      expect(attr(".rotor", "transform")).toBe("rotate(90 3 3) translate(1 1)");
    });
  });
});
