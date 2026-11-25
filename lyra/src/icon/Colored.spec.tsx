// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Icon } from "@/icon";

const renderSVG = (el: ReactElement): SVGSVGElement => {
  const svg = render(el).container.querySelector("svg");
  if (svg == null) throw new Error("expected the icon to render an svg");
  return svg;
};

const fills = (svg: SVGSVGElement): (string | null)[] =>
  Array.from(svg.querySelectorAll("g")).map((g) => g.getAttribute("fill"));

describe("Colored", () => {
  describe("brand logos", () => {
    it("should take the text color outside a colored region", () => {
      const svg = renderSVG(<Icon.Logo.Okta />);
      expect(svg.getAttribute("class")).not.toContain(CSS.symbolColored);
      expect(svg.style.color).toEqual("");
    });

    it("should take the brand color inside a colored region", () => {
      const svg = renderSVG(
        <Icon.Colored>
          <Icon.Logo.Okta className="custom" />
        </Icon.Colored>,
      );
      expect(svg.getAttribute("class")).toContain(CSS.symbolColored);
      expect(svg.getAttribute("class")).toContain("custom");
      expect(svg.getAttribute("class")).toContain(CSS.BM("icon", "logo-okta"));
      expect(svg.style.getPropertyValue("--pluto-symbol-color")).toEqual("0, 125, 193");
      expect(svg.style.color).toEqual("var(--pluto-symbol-display)");
    });

    it("should keep the text color for a brand with no color", () => {
      const svg = renderSVG(
        <Icon.Colored>
          <Icon.Logo.Apple />
        </Icon.Colored>,
      );
      expect(svg.getAttribute("class")).not.toContain(CSS.symbolColored);
      expect(svg.style.color).toEqual("");
    });
  });

  describe("multicolor logos", () => {
    it("should draw one color outside a colored region", () => {
      const svg = renderSVG(<Icon.Logo.Microsoft />);
      expect(svg.getAttribute("fill")).toEqual("currentColor");
      expect(fills(svg)).toEqual([null, null, null, null]);
    });

    it("should draw every brand color inside a colored region", () => {
      const svg = renderSVG(
        <Icon.Colored>
          <Icon.Logo.Microsoft />
        </Icon.Colored>,
      );
      expect(fills(svg)).toEqual(["#F25022", "#7FBA00", "#00A4EF", "#FFB900"]);
    });
  });
});
