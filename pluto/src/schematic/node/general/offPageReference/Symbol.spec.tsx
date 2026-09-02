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
import { describe, expect, it, vi } from "vitest";

import { type Config } from "@/schematic/node/general/offPageReference/config";
import { OffPageReference } from "@/schematic/node/general/offPageReference/Primitive";
import { Symbol } from "@/schematic/node/general/offPageReference/Symbol";
import { Theming } from "@/theming";

const ThemeWrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Theming.Provider>{children}</Theming.Provider>
);

describe("OffPageReference", () => {
  describe("color CSS variables", () => {
    it("should set the source color var and carry the symbol-colored class", () => {
      // The --off-page-color/--off-page-text-color vars are mapped to the display/
      // contrast vars in offPageReference.css; jsdom cannot compute them, so we assert
      // the source var (the only dynamic value) and the marker class.
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference color="#3774d0" />
        </ThemeWrapper>,
      );
      const arrow = container.querySelector<HTMLElement>(".pluto-arrow");
      expect(arrow?.getAttribute("class")).toContain("pluto-symbol-colored");
      // The source var carries the alpha channel so transparency survives the transform.
      expect(arrow?.style.getPropertyValue("--pluto-symbol-color")).toBe(
        `${color.rgbString("#3774d0")}, ${color.aValue("#3774d0")}`,
      );
    });

    it("should leave the source color unset for a default reference", () => {
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference />
        </ThemeWrapper>,
      );
      const arrow = container.querySelector<HTMLElement>(".pluto-arrow");
      expect(arrow?.style.getPropertyValue("--pluto-symbol-color")).toBe("");
    });

    it("should treat the ZERO default config color as unset", () => {
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference color={color.ZERO} />
        </ThemeWrapper>,
      );
      const arrow = container.querySelector<HTMLElement>(".pluto-arrow");
      expect(arrow?.style.getPropertyValue("--pluto-symbol-color")).toBe("");
    });
  });

  describe("page type icon", () => {
    it("should render the page type's icon when linked", () => {
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference linked pageType="lineplot" />
        </ThemeWrapper>,
      );
      expect(container.querySelector(".pluto-icon--line-plot")).not.toBeNull();
    });

    it("should render the schematic icon by default when linked", () => {
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference linked />
        </ThemeWrapper>,
      );
      expect(container.querySelector(".pluto-icon--schematic")).not.toBeNull();
    });

    it("should not render an icon when not linked", () => {
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference pageType="lineplot" />
        </ThemeWrapper>,
      );
      expect(container.querySelector(".pluto-icon")).toBeNull();
    });
  });

  describe("linked fill state", () => {
    it("should add the linked class when linked", () => {
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference linked />
        </ThemeWrapper>,
      );
      expect(container.querySelector(".pluto-arrow")?.getAttribute("class")).toContain(
        "pluto--linked",
      );
    });

    it("should not add the linked class when not linked", () => {
      const { container } = render(
        <ThemeWrapper>
          <OffPageReference />
        </ThemeWrapper>,
      );
      const cls = container.querySelector(".pluto-arrow")?.getAttribute("class") ?? "";
      expect(cls.includes("pluto--linked")).toBe(false);
    });
  });
});

describe("Symbol", () => {
  const renderSymbol = (page: Config["page"]) =>
    render(
      <ThemeWrapper>
        <Symbol
          nodeKey="n1"
          selected={false}
          onConfigChange={vi.fn()}
          config={{
            variant: "offPageReference",
            label: { label: "Ref" },
            page,
          }}
        />
      </ThemeWrapper>,
    );

  it("should render linked with the page type's icon for a typed page", () => {
    const { container } = renderSymbol({ type: "lineplot", key: "abc" });
    expect(container.querySelector(".pluto-icon--line-plot")).not.toBeNull();
    expect(container.querySelector(".pluto-arrow")?.getAttribute("class")).toContain(
      "pluto--linked",
    );
  });

  it("should render the schematic icon for a legacy bare page string", () => {
    const { container } = renderSymbol("abc");
    expect(container.querySelector(".pluto-icon--schematic")).not.toBeNull();
  });

  it("should render unlinked with no icon when the page is empty", () => {
    const { container } = renderSymbol("");
    expect(container.querySelector(".pluto-icon")).toBeNull();
    const cls = container.querySelector(".pluto-arrow")?.getAttribute("class") ?? "";
    expect(cls.includes("pluto--linked")).toBe(false);
  });
});
