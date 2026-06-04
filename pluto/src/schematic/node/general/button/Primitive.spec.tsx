// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Button } from "@/schematic/node/general/button/Primitive";

const getButton = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(".pluto-btn");
  if (el == null) throw new Error("expected a button element");
  return el;
};

describe("button symbol", () => {
  it("should drive its background, border, and text off the symbol display vars", () => {
    const { container } = render(<Button color="#ff0000" />);
    const btn = getButton(container);
    expect(btn.getAttribute("class")).toContain("pluto-symbol-colored");
    expect(btn.style.getPropertyValue("--pluto-symbol-color")).toBe("255, 0, 0, 1");
    expect(btn.style.getPropertyValue("--pluto-bg")).toBe(
      "var(--pluto-symbol-display)",
    );
    expect(btn.style.getPropertyValue("--pluto-btn-text-color")).toBe(
      "var(--pluto-symbol-contrast)",
    );
  });

  it("should not engage the base button's concrete-color JS path", () => {
    const { container } = render(<Button color="#ff0000" />);
    const btn = getButton(container);
    // The color is not forwarded, so the base button never sets its own color var.
    expect(btn.getAttribute("class")).not.toContain("pluto-btn--custom-color");
    expect(btn.style.getPropertyValue("--pluto-btn-color")).toBe("");
  });

  it("should carry the alpha channel so a translucent button stays translucent", () => {
    const { container } = render(<Button color={[255, 0, 0, 0.5]} />);
    expect(getButton(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
      "255, 0, 0, 0.5",
    );
  });

  it("should leave the source color unset for the ZERO sentinel", () => {
    const { container } = render(<Button color={color.ZERO} />);
    expect(getButton(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
      "",
    );
  });
});
