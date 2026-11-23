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
import { afterEach, describe, expect, it } from "vitest";
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

// createFrames builds a loop whose frames run only when a spec calls frame.
const createFrames = () => {
  let callback: FrameRequestCallback | null = null;
  const loop = new Custom.Loop({
    requestFrame: (cb) => {
      callback = cb;
      return 1;
    },
    cancelFrame: () => {
      callback = null;
    },
  });
  const frame = (now: number): void => {
    const cb = callback;
    callback = null;
    cb?.(now);
  };
  return { loop, frame };
};

interface HarnessProps extends Partial<Custom.DrawingProps> {
  loop: Custom.Loop;
}

const Harness = ({ loop, spec, ...rest }: HarnessProps): ReactElement => (
  <Custom.MotionProvider loop={loop}>
    <Primitive.Div>
      <Custom.Drawing spec={spec ?? createSpec([SPIN])} {...rest} />
    </Primitive.Div>
  </Custom.MotionProvider>
);

const setup = async (props: Partial<Custom.DrawingProps> = {}) => {
  const { loop, frame } = createFrames();
  const result = await render(<Harness loop={loop} {...props} />);
  const attr = (selector: string, name: string): string | null =>
    result.container.querySelector(selector)?.getAttribute(name) ?? null;
  const rerender = (next: Partial<Custom.DrawingProps>) =>
    result.rerender(<Harness loop={loop} {...props} {...next} />);
  return { ...result, loop, frame, attr, rerender };
};

const reading = (value: number, stale = false): Custom.Reading => ({ value, stale });

describe("Custom.Drawing motion", () => {
  afterEach(async () => {
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

  describe("spin", () => {
    it("should turn a part around the center of its own box at its rate", async () => {
      const { loop, frame, attr } = await setup();
      expect(loop.size).toBe(1);
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
      const { loop, frame, attr, rerender } = await setup();
      frame(0);
      frame(500);
      await rerender({ spec: createSpec([flow]) });
      expect(attr(".rotor", "transform")).toBe("translate(1 1)");
      expect(loop.size).toBe(1);
    });

    it("should read bounds that the Core refuses as fraction 0", async () => {
      const look = {
        ...SPIN,
        motion: { type: "spin" as const, rate: 1, state: "speed" },
      };
      const valid = createSpec([look]);
      const states = [{ ...STATES[0], bounds: { lower: 50, upper: 50 } }, STATES[1]];
      const { loop, frame, attr, rerender } = await setup({
        spec: { ...valid, states },
        readings: { speed: reading(50) },
      });
      expect(loop.size).toBe(0);
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
      const { loop, rerender } = await setup({ spec, readings: { speed: reading(0) } });
      expect(loop.size).toBe(0);
      await rerender({ readings: { speed: reading(50) } });
      expect(loop.size).toBe(1);
    });

    it("should stop and keep its angle while its state is stale", async () => {
      const look = {
        ...SPIN,
        motion: { type: "spin" as const, rate: 1, state: "speed" },
      };
      const spec = createSpec([look]);
      const { loop, frame, attr, rerender } = await setup({
        spec,
        readings: { speed: reading(100) },
      });
      frame(0);
      frame(250);
      await rerender({ readings: { speed: reading(100, true) } });
      expect(loop.size).toBe(0);
      expect(attr(".rotor", "transform")).toBe("rotate(90 3 3) translate(1 1)");
    });

    it("should stop while its state is unknown", async () => {
      const look = {
        ...SPIN,
        motion: { type: "spin" as const, rate: 1, state: "speed" },
      };
      const { loop } = await setup({
        spec: createSpec([look]),
        readings: { speed: reading(NaN) },
      });
      expect(loop.size).toBe(0);
    });

    it("should stop while a state that its look reads in when is stale", async () => {
      const look = {
        ...SPIN,
        when: { type: "equals" as const, state: "active", values: [1] },
      };
      const spec = createSpec([look]);
      const { loop, rerender } = await setup({
        spec,
        readings: { active: reading(1) },
      });
      expect(loop.size).toBe(1);
      await rerender({ readings: { active: reading(1, true) } });
      expect(loop.size).toBe(0);
    });

    it("should leave the loop and draw as drawn when its look stops", async () => {
      const look = {
        ...SPIN,
        when: { type: "equals" as const, state: "active", values: [1] },
      };
      const spec = createSpec([look]);
      const { loop, frame, attr, rerender } = await setup({
        spec,
        readings: { active: reading(1) },
      });
      frame(0);
      frame(500);
      await rerender({ readings: { active: reading(0) } });
      expect(loop.size).toBe(0);
      expect(attr(".rotor", "transform")).toBe("translate(1 1)");
    });

    it("should leave the loop while a later look hides the part", async () => {
      const hide: schematic.symbol.Look = { key: "hide", part: "rotor", hidden: true };
      const { loop } = await setup({ spec: createSpec([SPIN, hide]) });
      expect(loop.size).toBe(0);
    });

    it("should stop on a node with motion disabled", async () => {
      const { loop, rerender } = await setup();
      await rerender({ motionDisabled: true });
      expect(loop.size).toBe(0);
    });

    it("should stop under the Reduce motion setting", async () => {
      const { loop } = await setup();
      expect(loop.size).toBe(1);
      await commands.emulateReducedMotion(true);
      await expect.poll(() => loop.size).toBe(0);
    });

    it("should leave the loop when the drawing is removed", async () => {
      const { loop, unmount } = await setup();
      await unmount();
      expect(loop.size).toBe(0);
    });

    it("should move the new drawing when the markup changes", async () => {
      const spec = createSpec([SPIN]);
      const { loop, frame, attr, rerender } = await setup({ spec });
      await rerender({
        spec: { ...spec, drawing: { type: "svg", markup: MARKUP.replace("4", "3") } },
      });
      expect(loop.size).toBe(1);
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
      const { loop, frame, attr, rerender } = await setup({
        spec,
        readings: { active: reading(1) },
      });
      frame(0);
      frame(600);
      await rerender({ readings: { active: reading(1, true) } });
      expect(loop.size).toBe(0);
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
      const { loop, attr } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
      });
      expect(loop.size).toBe(0);
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
      const { loop, frame, attr, rerender } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
      });
      await rerender({ readings: { speed: reading(75) } });
      expect(loop.size).toBe(1);
      frame(0);
      frame(100);
      expect(attr(".liquid", "clip-path")).not.toBe("inset(25% 0 0 0)");
      frame(200);
      expect(attr(".liquid", "clip-path")).toBe("inset(25% 0 0 0)");
      expect(loop.size).toBe(0);
    });

    it("should jump to a new fraction on a node with motion disabled", async () => {
      const { loop, attr, rerender } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
        motionDisabled: true,
      });
      await rerender({ readings: { speed: reading(75) } });
      expect(loop.size).toBe(0);
      expect(attr(".liquid", "clip-path")).toBe("inset(25% 0 0 0)");
    });

    it("should draw an empty level while its state is unknown", async () => {
      const { loop, attr, rerender } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
      });
      await rerender({ readings: { speed: reading(NaN) } });
      expect(loop.size).toBe(0);
      expect(attr(".liquid", "clip-path")).toBe("inset(100% 0 0 0)");
    });

    it("should keep its last fraction while its state is stale", async () => {
      const { loop, attr, rerender } = await setup({
        spec: level(),
        readings: { speed: reading(25) },
      });
      await rerender({ readings: { speed: reading(75, true) } });
      expect(loop.size).toBe(0);
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
});
