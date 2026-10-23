// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ControlSystemGraphic } from "@/components/automate/ControlSystemGraphic";

const keyframes = (svg: SVGSVGElement): string[] =>
  [...svg.querySelector("style")!.textContent.matchAll(/@keyframes (\S+)/g)].map(
    (m) => m[1],
  );

describe("ControlSystemGraphic", () => {
  it("should label three rows of twelve ticks", () => {
    const { container } = render(<ControlSystemGraphic />);
    const labels = [...container.querySelectorAll("text")].map((t) => t.textContent);
    expect(labels).toEqual(["Auto 0", "Auto 1", "Auto 2"]);
    expect(container.querySelectorAll('circle[r="4"]')).toHaveLength(36);
  });

  it("should be three quarters as tall as it is wide", () => {
    const { container } = render(<ControlSystemGraphic size={400} />);
    expect(container.querySelector("svg")!.getAttribute("viewBox")).toBe("0 0 400 300");
  });

  it("should name its animations with letters and digits alone", () => {
    const { container } = render(<ControlSystemGraphic />);
    const names = keyframes(container.querySelector("svg")!);
    expect(names).toHaveLength(3);
    for (const name of names) expect(name).toMatch(/^[a-z-]+[a-zA-Z0-9]+$/);
  });

  it("should give each graphic its own animations and gradients", () => {
    const { container } = render(
      <>
        <ControlSystemGraphic />
        <ControlSystemGraphic />
      </>,
    );
    const [a, b] = [...container.querySelectorAll("svg")];
    expect(keyframes(a)).not.toEqual(keyframes(b));
    for (const svg of [a, b]) {
      const names = keyframes(svg);
      const animations = [...svg.querySelectorAll<SVGElement>("[style*='animation']")];
      for (const el of animations)
        expect(names).toContain(el.style.animation.split(" ")[0]);
      for (const el of svg.querySelectorAll("[fill^='url(#']"))
        expect(svg.querySelector(el.getAttribute("fill")!.slice(4, -1))).not.toBeNull();
    }
  });
});
