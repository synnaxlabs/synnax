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

import { Icon } from "@/icon";
import { Theming } from "@/theming";

const renderSVG = (el: ReactElement, dark = false): SVGSVGElement => {
  const svg = render(
    <Theming.Provider theme={{ key: dark ? "synnaxDark" : "synnaxLight" }}>
      {el}
    </Theming.Provider>,
  ).container.querySelector("svg");
  if (svg == null) throw new Error("expected the icon to render an svg");
  return svg;
};

const partColors = (svg: SVGSVGElement): string[] =>
  Array.from(svg.querySelectorAll("g")).map((g) => g.style.color);

describe("Colored", () => {
  describe("brand logos", () => {
    it("should take the text color outside a colored region", () => {
      const svg = renderSVG(<Icon.Logo.LabJack />);
      expect(svg.style.color).toEqual("");
    });

    it("should take the brand's light color in the light theme", () => {
      const svg = renderSVG(
        <Icon.Colored>
          <Icon.Logo.LabJack className="custom" />
        </Icon.Colored>,
      );
      expect(svg.getAttribute("class")).toContain("custom");
      expect(svg.style.color).toEqual("rgb(141, 2, 0)");
    });

    it("should take the brand's dark color in the dark theme", () => {
      const svg = renderSVG(
        <Icon.Colored>
          <Icon.Logo.LabJack />
        </Icon.Colored>,
        true,
      );
      expect(svg.style.color).toEqual("rgb(226, 96, 80)");
    });

    it("should keep the text color in a theme the brand draws it white", () => {
      const logo = (
        <Icon.Colored>
          <Icon.Logo.Keycloak />
        </Icon.Colored>
      );
      expect(renderSVG(logo).style.color).toEqual("rgb(0, 184, 227)");
      expect(renderSVG(logo, true).style.color).toEqual("");
    });

    it("should keep the text color for a brand with no color", () => {
      const svg = renderSVG(
        <Icon.Colored>
          <Icon.Logo.Apple />
        </Icon.Colored>,
      );
      expect(svg.style.color).toEqual("");
    });
  });

  describe("multicolor logos", () => {
    const microsoft = [
      "rgb(242, 80, 34)",
      "rgb(127, 186, 0)",
      "rgb(0, 164, 239)",
      "rgb(255, 185, 0)",
    ];

    it("should draw one color outside a colored region", () => {
      const svg = renderSVG(<Icon.Logo.Microsoft />);
      expect(svg.getAttribute("fill")).toEqual("currentColor");
      expect(partColors(svg)).toEqual(["", "", "", ""]);
    });

    it("should draw each part in its official color in both themes", () => {
      for (const dark of [false, true]) {
        const svg = renderSVG(
          <Icon.Colored>
            <Icon.Logo.Microsoft />
          </Icon.Colored>,
          dark,
        );
        expect(partColors(svg)).toEqual(microsoft);
        expect(svg.getAttribute("stroke-width")).toEqual("0");
        for (const g of svg.querySelectorAll("g"))
          expect(g.getAttribute("fill")).toEqual("currentColor");
      }
    });

    it("should color only the brand parts of a mark", () => {
      const svg = renderSVG(
        <Icon.Colored>
          <Icon.Logo.EtherCAT />
        </Icon.Colored>,
      );
      expect(partColors(svg)).toEqual(["rgb(237, 28, 36)"]);
      expect(svg.querySelectorAll(":scope > path")).toHaveLength(3);
    });
  });
});
