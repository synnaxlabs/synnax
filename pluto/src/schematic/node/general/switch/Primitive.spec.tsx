// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Switch } from "@/schematic/node/general/switch/Primitive";

const getRoot = (container: HTMLElement): HTMLElement => {
  const el = container.firstElementChild;
  if (el == null) throw new Error("expected a switch element");
  return el as HTMLElement;
};

const getSVG = (container: HTMLElement): SVGSVGElement => {
  const el = container.querySelector<SVGSVGElement>("svg");
  if (el == null) throw new Error("expected a switch drawing");
  return el;
};

const classOf = (el: Element): string => el.getAttribute("class") ?? "";

describe("switch symbol", () => {
  it("should draw a track and a knob", () => {
    const { container } = render(<Switch />);
    expect(container.querySelector(".pluto-switch-symbol__track")).not.toBeNull();
    expect(container.querySelector(".pluto-switch-symbol__knob")).not.toBeNull();
  });

  describe("color", () => {
    // switch.css paints the track and knob from the display var, which jsdom cannot
    // compute, so these specs assert the classes and the source var.
    it("should mark a colored switch and set the source color on its drawing", () => {
      const { container } = render(<Switch strokeColor="#ff0000" />);
      expect(classOf(getRoot(container))).toContain("pluto-switch-symbol--colored");
      const svg = getSVG(container);
      expect(classOf(svg)).toContain("lyra-symbol-colored");
      expect(svg.style.getPropertyValue("--pluto-symbol-color")).toBe("255, 0, 0, 1");
    });

    it("should carry the alpha channel so a translucent color stays translucent", () => {
      const { container } = render(<Switch strokeColor={[255, 0, 0, 51]} />);
      expect(getSVG(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
        "255, 0, 0, 0.2",
      );
    });

    it("should pass a fully transparent color through as a choice", () => {
      const { container } = render(<Switch strokeColor={color.TRANSPARENT} />);
      expect(classOf(getRoot(container))).toContain("pluto-switch-symbol--colored");
      expect(getSVG(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
        "0, 0, 0, 0",
      );
    });

    it("should stay uncolored when no color is given", () => {
      const { container } = render(<Switch />);
      expect(classOf(getRoot(container))).not.toContain("pluto-switch-symbol--colored");
    });
  });

  describe("scale", () => {
    // switch.css caps the gap around the knob from the scale var.
    it("should set the scale var from the scale prop", () => {
      const { container } = render(<Switch scale={3} />);
      expect(getRoot(container).style.getPropertyValue("--pluto-switch-scale")).toBe(
        "3",
      );
    });

    it.each([0, -2, NaN])("should render a scale of %s at 1", (scale) => {
      const { container } = render(<Switch scale={scale} />);
      expect(getRoot(container).style.getPropertyValue("--pluto-switch-scale")).toBe(
        "1",
      );
    });
  });

  describe("orientation", () => {
    it.each([
      ["left", "left"],
      ["right", "left"],
      ["top", "top"],
      ["bottom", "top"],
    ] as const)("should draw a switch facing %s as one facing %s", (side, drawn) => {
      const { container } = render(<Switch orientation={side} />);
      expect(classOf(getSVG(container))).toContain(`lyra--location-${drawn}`);
    });
  });

  describe("enabled", () => {
    it("should mark an enabled switch as on", () => {
      const { container } = render(<Switch enabled />);
      expect(classOf(getRoot(container))).toContain("pluto-switch-symbol--on");
    });

    it("should default to off", () => {
      const { container } = render(<Switch />);
      expect(classOf(getRoot(container))).not.toContain("pluto-switch-symbol--on");
    });

    it("should call onClick when clicked", () => {
      const onClick = vi.fn();
      const { container } = render(<Switch onClick={onClick} />);
      fireEvent.click(getRoot(container));
      expect(onClick).toHaveBeenCalledOnce();
    });

    it("should not call onClick while disabled", () => {
      const onClick = vi.fn();
      const { container } = render(<Switch onClick={onClick} disabled />);
      expect(classOf(getRoot(container))).toContain("pluto-switch-symbol--disabled");
      fireEvent.click(getRoot(container));
      expect(onClick).not.toHaveBeenCalled();
    });
  });
});
