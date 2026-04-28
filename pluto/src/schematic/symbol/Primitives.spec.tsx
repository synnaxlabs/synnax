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
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { OffPageReference } from "@/schematic/symbol/Primitives";
import { Theming } from "@/theming";

const ThemeWrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Theming.Provider>{children}</Theming.Provider>
);

const getOutline = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(".outline");
  if (el == null) throw new Error("expected .outline element to exist");
  return el;
};

describe("OffPageReference", () => {
  describe("color CSS variables", () => {
    it("should set --off-page-color from the color prop", () => {
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference color="#3774d0" />
        </ThemeWrapper>,
      );
      const outline = getOutline(container);
      expect(outline.style.getPropertyValue("--off-page-color")).toBe(
        color.cssString("#3774d0"),
      );
    });

    it("should pick light text for a dark background", () => {
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference color="#000000" />
        </ThemeWrapper>,
      );
      const outline = getOutline(container);
      const textColor = outline.style.getPropertyValue("--off-page-text-color");
      // Black background should produce a light (high-luminance) text color
      // The theme will pick textInverted (light) over text (dark)
      expect(textColor).not.toBe("");
      expect(textColor).not.toBe(color.cssString("#000000"));
    });

    it("should pick dark text for a light background", () => {
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference color="#FFFFFF" />
        </ThemeWrapper>,
      );
      const outline = getOutline(container);
      const textColor = outline.style.getPropertyValue("--off-page-text-color");
      // White background should produce a dark text color
      expect(textColor).not.toBe("");
      expect(textColor).not.toBe(color.cssString("#FFFFFF"));
    });

    it("should produce different text colors for dark vs light backgrounds", () => {
      const { container: darkContainer } = render(
        <ThemeWrapper>
          <OffPageReference color="#000000" />
        </ThemeWrapper>,
      );
      const { container: lightContainer } = render(
        <ThemeWrapper>
          <OffPageReference color="#FFFFFF" />
        </ThemeWrapper>,
      );
      const darkTextColor = getOutline(darkContainer).style.getPropertyValue(
        "--off-page-text-color",
      );
      const lightTextColor = getOutline(lightContainer).style.getPropertyValue(
        "--off-page-text-color",
      );
      expect(darkTextColor).not.toBe(lightTextColor);
    });

    it("should use a default color when color prop is not provided", () => {
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference />
        </ThemeWrapper>,
      );
      const outline = getOutline(container);
      expect(outline.style.getPropertyValue("--off-page-color")).not.toBe("");
      expect(outline.style.getPropertyValue("--off-page-text-color")).not.toBe("");
    });
  });
});
