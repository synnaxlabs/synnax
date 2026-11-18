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

import { CSS } from "@/css";
import { Icon } from "@/icon";

const renderSVG = (props: Icon.IconProps): SVGSVGElement => {
  const { container } = render(<Icon.Valve {...props} />);
  const svg = container.querySelector("svg");
  if (svg == null) throw new Error("expected the icon to render an svg");
  return svg;
};

describe("Icon", () => {
  describe("createFromPath", () => {
    it("should fill the path in the icon's color under the icon's name", () => {
      const Glyph = Icon.createFromPath("glyph", "M2,2h20v20h-20z");
      const { container } = render(<Glyph color="#dc136c" />);
      const svg = container.querySelector("svg");
      expect(svg?.getAttribute("class")).toContain(CSS.BM("icon", "glyph"));
      expect(svg?.getAttribute("viewBox")).toEqual("0 0 24 24");
      expect(svg?.getAttribute("fill")).toEqual("currentColor");
      expect(svg?.getAttribute("color")).toEqual("#dc136c");
      expect(container.querySelector("path")?.getAttribute("d")).toEqual(
        "M2,2h20v20h-20z",
      );
    });
  });

  describe("tint", () => {
    it("should leave the icon's color alone without a color", () => {
      const svg = renderSVG(Icon.tint());
      expect(svg.getAttribute("class")).not.toContain(CSS.symbolColored);
      expect(svg.style.color).toEqual("");
    });

    it("should draw the icon through the icon band of the theme", () => {
      const svg = renderSVG(Icon.tint("#dc136c"));
      expect(svg.getAttribute("class")).toContain(CSS.symbolColored);
      expect(svg.getAttribute("class")).toContain(CSS.BM("symbol-colored", "icon"));
      expect(svg.style.getPropertyValue("--pluto-symbol-color")).toEqual(
        "220, 19, 108",
      );
      expect(svg.style.color).toEqual("var(--pluto-symbol-display)");
    });

    it("should drop the alpha of a translucent color", () => {
      const svg = renderSVG(Icon.tint("#dc136c40"));
      expect(svg.style.getPropertyValue("--pluto-symbol-color")).toEqual(
        "220, 19, 108",
      );
    });
  });

  describe("Dot", () => {
    it("should put its color on its svg, as the other icons do", () => {
      const { container } = render(
        <>
          <Icon.Dot color="#3774d0" />
          <Icon.Circle color="#3774d0" />
        </>,
      );
      const [dot, circle] = container.querySelectorAll("svg");
      expect(dot.getAttribute("color")).not.toBeNull();
      expect(dot.getAttribute("color")).toEqual(circle.getAttribute("color"));
    });
  });

  describe("resolve", () => {
    it("should render the icon at a path in the registry", () => {
      const { container } = render(Icon.resolve("Arrow.Up"));
      expect(container.querySelector("svg")?.getAttribute("class")).toContain(
        CSS.BM("icon", "arrow-up"),
      );
    });

    it("should throw for a path the registry does not hold", () => {
      expect(() => Icon.resolve("Arrow.Sideways")).toThrow(
        "Unable to find icon with path Arrow.Sideways in registry",
      );
    });

    it("should throw for a path under a group the registry does not hold", () => {
      expect(() => Icon.resolve("Missing.Up")).toThrow(
        "Unable to find icon with path Missing.Up in registry",
      );
    });

    it("should return undefined for no icon", () => {
      expect(Icon.resolve()).toBeUndefined();
    });
  });
});
