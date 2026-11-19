// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Input } from "@synnaxlabs/lyra/input";
import { computed, resolve } from "@synnaxlabs/lyra/styletest";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color, TimeSpan } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { render } from "vitest-browser-react";

import { Switch } from "@/schematic/node/general/switch/Primitive";

interface Geometry {
  trackWidth: number;
  trackHeight: number;
  knobSize: number;
  knobLeft: number;
  knobTop: number;
}

const find = (container: HTMLElement, selector: string): Element => {
  const el = container.querySelector(selector);
  if (el == null) throw new Error(`expected ${selector}`);
  return el;
};

const px = (value: string): number => parseFloat(value);

const renderInTheme = async (children: ReactElement): Promise<HTMLElement> => {
  const { container } = await render(
    <Theming.Provider theme={Theming.SYNNAX_LIGHT}>{children}</Theming.Provider>,
  );
  return container;
};

// The knob is the indicator's ::before, which has no box of its own to measure.
const measureLyra = async (scale: number, enabled: boolean): Promise<Geometry> => {
  const style = { "--pluto-switch-scale": scale } as CSSProperties;
  const container = await renderInTheme(
    <div style={style}>
      <Input.Switch value={enabled} onChange={() => {}} />
    </div>,
  );
  const track = find(container, ".lyra-input__switch-indicator");
  const { width, height } = track.getBoundingClientRect();
  const knob = getComputedStyle(track, "::before");
  const knobSize = px(knob.width);
  const travel = enabled ? new DOMMatrix(knob.transform).m41 : 0;
  return {
    trackWidth: width,
    trackHeight: height,
    knobSize,
    knobLeft: px(knob.left) + travel,
    knobTop: height - px(knob.bottom) - knobSize,
  };
};

const measureSymbol = async (scale: number, enabled: boolean): Promise<Geometry> => {
  const container = await renderInTheme(<Switch scale={scale} enabled={enabled} />);
  const track = find(container, ".pluto-switch-symbol__track").getBoundingClientRect();
  const knob = find(container, ".pluto-switch-symbol__knob").getBoundingClientRect();
  return {
    trackWidth: track.width,
    trackHeight: track.height,
    knobSize: knob.width,
    knobLeft: knob.left - track.left,
    knobTop: knob.top - track.top,
  };
};

interface Fills {
  track: string;
  knob: string;
}

const lyraFills = async (enabled: boolean, disabled: boolean): Promise<Fills> => {
  const container = await renderInTheme(
    <Input.Switch value={enabled} disabled={disabled} onChange={() => {}} />,
  );
  const track = find(container, ".lyra-input__switch-indicator");
  return {
    track: computed(track, "background-color"),
    knob: getComputedStyle(track, "::before").getPropertyValue("background-color"),
  };
};

const COLOR = color.construct("#3e8ef7");

const renderFills = async (
  props: { enabled?: boolean; disabled?: boolean; strokeColor?: color.Color } = {},
): Promise<Fills & { at: (value: string) => string }> => {
  const container = await renderInTheme(<Switch {...props} />);
  const drawing = find(container, ".pluto-switch-symbol svg");
  // A probe shape inside the drawing resolves a fill against its own variables.
  const at = (value: string): string => {
    const probe = document.createElementNS("http://www.w3.org/2000/svg", "rect");
    probe.style.setProperty("fill", value);
    drawing.appendChild(probe);
    const fill = computed(probe, "fill");
    probe.remove();
    return fill;
  };
  return {
    track: computed(find(container, ".pluto-switch-symbol__track"), "fill"),
    knob: computed(find(container, ".pluto-switch-symbol__knob"), "fill"),
    at,
  };
};

const releaseHold = (): void => {
  document.dispatchEvent(new MouseEvent("mouseup"));
};

describe("switch symbol in a browser", () => {
  afterEach(releaseHold);

  describe("geometry", () => {
    it.each([1, 2, 3])(
      "should size and place the knob as the Lyra switch does at scale %s",
      async (scale) => {
        for (const enabled of [false, true]) {
          const lyra = await measureLyra(scale, enabled);
          const symbol = await measureSymbol(scale, enabled);
          for (const key of Object.keys(lyra) as Array<keyof Geometry>)
            expect(symbol[key], `${key}, enabled ${enabled}`).toBeCloseTo(lyra[key], 1);
        }
      },
    );

    it.each([
      [false, "below"],
      [true, "above"],
    ] as const)(
      "should stand upright with an enabled of %s knob %s the center",
      async (enabled, side) => {
        const container = await renderInTheme(
          <Switch orientation="top" enabled={enabled} />,
        );
        const track = find(
          container,
          ".pluto-switch-symbol__track",
        ).getBoundingClientRect();
        const knob = find(
          container,
          ".pluto-switch-symbol__knob",
        ).getBoundingClientRect();
        const lyra = await measureLyra(1, enabled);
        expect(track.width).toBeCloseTo(lyra.trackHeight, 1);
        expect(track.height).toBeCloseTo(lyra.trackWidth, 1);
        expect(knob.left + knob.width / 2).toBeCloseTo(track.left + track.width / 2, 1);
        const knobCenter = knob.top + knob.height / 2;
        const trackCenter = track.top + track.height / 2;
        if (side === "below") expect(knobCenter).toBeGreaterThan(trackCenter);
        else expect(knobCenter).toBeLessThan(trackCenter);
      },
    );
  });

  describe("fills", () => {
    it.each([
      [false, false],
      [true, false],
      [false, true],
      [true, true],
    ])(
      "should fill an enabled of %s, disabled of %s switch as the Lyra switch does",
      async (enabled, disabled) => {
        const { track, knob } = await renderFills({ enabled, disabled });
        expect({ track, knob }).toEqual(await lyraFills(enabled, disabled));
      },
    );

    it("should fill an off colored switch with a tint of its color", async () => {
      const { track, knob, at } = await renderFills({ strokeColor: COLOR });
      expect(knob).toBe(at("var(--pluto-symbol-display)"));
      expect(track).toBe(at("oklch(from var(--pluto-symbol-display) l c h / 25%)"));
    });

    it("should fill an on colored switch with its color", async () => {
      const { track, knob, at } = await renderFills({
        enabled: true,
        strokeColor: COLOR,
      });
      expect(track).toBe(at("var(--pluto-symbol-display)"));
      expect(knob).toBe(at("var(--pluto-gray-l1)"));
    });

    it.each([false, true])(
      "should fill a disabled colored switch with an enabled of %s as a disabled one",
      async (enabled) => {
        const { track, knob } = await renderFills({
          enabled,
          disabled: true,
          strokeColor: COLOR,
        });
        expect({ track, knob }).toEqual(await lyraFills(enabled, true));
      },
    );
  });

  describe("hold", () => {
    const DELAY = TimeSpan.seconds(60);

    const renderDelayed = async (
      props: { enabled?: boolean; strokeColor?: color.Color } = {},
    ): Promise<HTMLElement> =>
      await renderInTheme(
        <Switch onClickDelay={DELAY} onClick={() => {}} {...props} />,
      );

    const press = (container: HTMLElement): void => {
      find(container, ".pluto-switch-symbol").dispatchEvent(
        new MouseEvent("mousedown", { bubbles: true, button: 0 }),
      );
    };

    it("should hide the hold fill on a switch with no delay", async () => {
      const container = await renderInTheme(<Switch />);
      expect(computed(find(container, ".pluto-switch-symbol__hold"), "display")).toBe(
        "none",
      );
    });

    // The fill starts at the side the knob leaves, so it runs the way the knob moves.
    it.each([
      [false, "inset(0px 100% 0px 0px)"],
      [true, "inset(0px 0px 0px 100%)"],
    ])(
      "should start the fill of an enabled of %s switch at %s",
      async (enabled, clip) => {
        const container = await renderDelayed({ enabled });
        expect(
          computed(find(container, ".pluto-switch-symbol__hold"), "clip-path"),
        ).toBe(clip);
      },
    );

    it.each([false, true])(
      "should cover the track while an enabled of %s switch is held",
      async (enabled) => {
        const container = await renderDelayed({ enabled });
        press(container);
        await expect
          .poll(() =>
            computed(find(container, ".pluto-switch-symbol__hold"), "clip-path"),
          )
          .toBe("inset(0px)");
      },
    );

    it("should tint the fill only on a colored switch", async () => {
      const transparent = resolve("transparent");
      const tint = (container: HTMLElement): string =>
        computed(find(container, ".pluto-switch-symbol__tint"), "fill");
      expect(tint(await renderDelayed())).toBe(transparent);
      const colored = await renderDelayed({ strokeColor: COLOR });
      expect(tint(colored)).not.toBe(transparent);
    });
  });
});
