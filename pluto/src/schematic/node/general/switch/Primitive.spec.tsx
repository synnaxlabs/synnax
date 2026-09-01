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

const getInput = (container: HTMLElement): HTMLInputElement => {
  const el = container.querySelector<HTMLInputElement>("input");
  if (el == null) throw new Error("expected a switch input");
  return el;
};

describe("switch symbol", () => {
  describe("color", () => {
    // The track and knob are painted from the display var in switch.css; jsdom cannot
    // compute it, so we assert the marker classes and the source var.
    it("should carry the symbol-colored + colored classes and set the source color", () => {
      const { container } = render(<Switch color="#ff0000" />);
      const root = getRoot(container);
      const cls = root.getAttribute("class") ?? "";
      expect(cls).toContain("pluto-symbol-colored");
      expect(cls).toContain("pluto-switch-symbol--colored");
      expect(root.style.getPropertyValue("--pluto-symbol-color")).toBe("255, 0, 0, 1");
    });

    it("should carry the alpha channel so a translucent color stays translucent", () => {
      const { container } = render(<Switch color={[255, 0, 0, 0.5]} />);
      expect(getRoot(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
        "255, 0, 0, 0.5",
      );
    });

    // An unset color must leave the switch on the input theme it draws itself from.
    it("should stay uncolored for the ZERO sentinel", () => {
      const { container } = render(<Switch color={color.ZERO} />);
      const root = getRoot(container);
      const cls = root.getAttribute("class") ?? "";
      expect(cls).not.toContain("pluto-symbol-colored");
      expect(cls).not.toContain("pluto-switch-symbol--colored");
      expect(root.style.getPropertyValue("--pluto-symbol-color")).toBe("");
    });

    it("should stay uncolored when no color is given", () => {
      const { container } = render(<Switch />);
      expect(getRoot(container).getAttribute("class")).not.toContain(
        "pluto-switch-symbol--colored",
      );
    });

    // The stale color reaches the primitive as an ordinary color, so both states color.
    it("should color an enabled switch the same way as a disabled one", () => {
      const { container } = render(<Switch color="#ff0000" enabled />);
      const root = getRoot(container);
      expect(root.getAttribute("class")).toContain("pluto-switch-symbol--colored");
      expect(root.style.getPropertyValue("--pluto-symbol-color")).toBe("255, 0, 0, 1");
    });
  });

  describe("scale", () => {
    // The dimensions are computed from the scale var in Switch.css; jsdom cannot
    // compute them, so we assert the source var.
    it("should set the scale var from the scale prop", () => {
      const { container } = render(<Switch scale={3} />);
      expect(getRoot(container).style.getPropertyValue("--pluto-switch-scale")).toBe(
        "3",
      );
    });

    it("should default the scale var to 1", () => {
      const { container } = render(<Switch />);
      expect(getRoot(container).style.getPropertyValue("--pluto-switch-scale")).toBe(
        "1",
      );
    });
  });

  describe("orientation", () => {
    // The box swap and rotation live in switch.css; jsdom cannot compute them, so we
    // assert the location class that keys them.
    it("should carry the location class for the orientation prop", () => {
      const { container } = render(<Switch orientation="bottom" />);
      expect(getRoot(container).getAttribute("class")).toContain(
        "pluto--location-bottom",
      );
    });

    it("should default to the left location", () => {
      const { container } = render(<Switch />);
      expect(getRoot(container).getAttribute("class")).toContain(
        "pluto--location-left",
      );
    });
  });

  describe("enabled", () => {
    it("should reflect the enabled state on the input", () => {
      const { container } = render(<Switch enabled />);
      expect(getInput(container).checked).toBe(true);
    });

    it("should default to off", () => {
      const { container } = render(<Switch />);
      expect(getInput(container).checked).toBe(false);
    });

    it("should call onClick when clicked", () => {
      const onClick = vi.fn();
      const { container } = render(<Switch onClick={onClick} />);
      fireEvent.click(getInput(container));
      expect(onClick).toHaveBeenCalledOnce();
    });
  });
});
