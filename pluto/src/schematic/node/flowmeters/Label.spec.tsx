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

import { Label } from "@/schematic/node/flowmeters/Label";

const renderInSVG = (children: React.ReactNode) =>
  render(<svg data-testid="root">{children}</svg>);

const queryText = (container: HTMLElement): SVGTextElement | null =>
  container.querySelector("text");

describe("Label", () => {
  describe("position", () => {
    it("should default to the F-label anchor (57, 27)", () => {
      const { container } = renderInSVG(<Label />);
      const text = queryText(container);
      expect(text?.getAttribute("x")).toBe("57");
      expect(text?.getAttribute("y")).toBe("27");
    });

    it("should accept an explicit position override", () => {
      const { container } = renderInSVG(<Label position={{ x: 10, y: 20 }} />);
      const text = queryText(container);
      expect(text?.getAttribute("x")).toBe("10");
      expect(text?.getAttribute("y")).toBe("20");
    });
  });

  describe("color", () => {
    it("should fall back to the theme's default color when none is provided", () => {
      const { container } = renderInSVG(<Label />);
      const text = queryText(container) as SVGTextElement;
      expect(text.style.fill).toMatch(/^rgba?\(/);
    });

    it("should serialize a hex color into the fill style as rgba", () => {
      const { container } = renderInSVG(<Label color="#ff0000" />);
      const text = queryText(container) as SVGTextElement;
      // color.cssString turns hex into an rgba(...) CSS value.
      expect(text.style.fill).toMatch(/^rgba?\(/);
      expect(text.style.fill).toMatch(/255\s*,\s*0\s*,\s*0/);
    });
  });

  describe("static structure", () => {
    it("should render the letter F", () => {
      const { container } = renderInSVG(<Label />);
      expect(queryText(container)?.textContent).toBe("F");
    });

    it("should always set stroke to none", () => {
      const { container } = renderInSVG(<Label color="#00ff00" />);
      expect(queryText(container)?.getAttribute("stroke")).toBe("none");
    });
  });
});
