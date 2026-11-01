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
});
