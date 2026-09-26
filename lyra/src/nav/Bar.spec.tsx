// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Nav } from "@/nav";

const barOf = (c: ReturnType<typeof render>): HTMLElement => {
  const bar = c.container.querySelector<HTMLElement>(".pluto-navbar");
  if (bar == null) throw new Error("navbar not found");
  return bar;
};

const contentOf = (c: ReturnType<typeof render>): HTMLElement => {
  const content = c.container.querySelector<HTMLElement>(".pluto-navbar__content");
  if (content == null) throw new Error("navbar content not found");
  return content;
};

describe("Nav.Bar", () => {
  it("should render its children", () => {
    const c = render(
      <Nav.Bar>
        <p>Hello</p>
      </Nav.Bar>,
    );
    expect(c.getByText("Hello")).toBeTruthy();
  });

  it("should default to the left location", () => {
    const c = render(<Nav.Bar />);
    expect(barOf(c).className).toContain("pluto--location-left");
  });

  it("should forward an arbitrary class name", () => {
    const c = render(<Nav.Bar className="custom-bar" />);
    expect(barOf(c).className).toContain("custom-bar");
  });

  describe("location", () => {
    interface LocationCase {
      location: location.Crude;
      locClass: string;
      dirClass: string;
      dimension: "width" | "height";
    }
    const CASES: LocationCase[] = [
      {
        location: "left",
        locClass: "pluto--location-left",
        dirClass: "pluto--direction-y",
        dimension: "width",
      },
      {
        location: "right",
        locClass: "pluto--location-right",
        dirClass: "pluto--direction-y",
        dimension: "width",
      },
      {
        location: "top",
        locClass: "pluto--location-top",
        dirClass: "pluto--direction-x",
        dimension: "height",
      },
      {
        location: "bottom",
        locClass: "pluto--location-bottom",
        dirClass: "pluto--direction-x",
        dimension: "height",
      },
    ];
    it.each(CASES)(
      "should apply the location and opposite-direction classes for a $location bar",
      ({ location, locClass, dirClass }) => {
        const c = render(<Nav.Bar location={location} />);
        const bar = barOf(c);
        expect(bar.className).toContain(locClass);
        expect(bar.className).toContain(dirClass);
      },
    );

    it.each(CASES)(
      "should apply the size to the $dimension for a $location bar",
      ({ location, dimension }) => {
        const c = render(<Nav.Bar location={location} size="5rem" />);
        expect(barOf(c).style[dimension]).toEqual("5rem");
      },
    );
  });

  describe("size", () => {
    it("should default to a width of 9rem for a left bar", () => {
      const c = render(<Nav.Bar location="left" />);
      expect(barOf(c).style.width).toEqual("9rem");
    });

    it("should apply a string size verbatim", () => {
      const c = render(<Nav.Bar location="left" size="12rem" />);
      expect(barOf(c).style.width).toEqual("12rem");
    });

    it("should apply a numeric size as pixels", () => {
      const c = render(<Nav.Bar location="left" size={240} />);
      expect(barOf(c).style.width).toEqual("240px");
    });

    it("should merge a caller-provided style with the size", () => {
      const c = render(
        <Nav.Bar location="left" size="9rem" style={{ background: "red" }} />,
      );
      const bar = barOf(c);
      expect(bar.style.width).toEqual("9rem");
      expect(bar.style.background).toEqual("red");
    });
  });

  describe("bordered", () => {
    interface BorderedCase {
      location: location.Crude;
      borderClass: string;
    }
    it.each<BorderedCase>([
      { location: "left", borderClass: "pluto--bordered-right" },
      { location: "right", borderClass: "pluto--bordered-left" },
      { location: "top", borderClass: "pluto--bordered-bottom" },
      { location: "bottom", borderClass: "pluto--bordered-top" },
    ])(
      "should add a border on the swapped side for a $location bar",
      ({ location, borderClass }) => {
        const c = render(<Nav.Bar location={location} bordered />);
        expect(barOf(c).className).toContain(borderClass);
      },
    );

    it("should not add a border class by default", () => {
      const c = render(<Nav.Bar location="left" />);
      expect(barOf(c).className).not.toContain("pluto--bordered");
    });
  });

  describe("content", () => {
    interface ContentCase {
      name: string;
      element: ReactElement;
      alignClass: string;
    }
    it.each<ContentCase>([
      {
        name: "Start",
        element: (
          <Nav.Bar.Start>
            <p>x</p>
          </Nav.Bar.Start>
        ),
        alignClass: "pluto--start",
      },
      {
        name: "Center",
        element: (
          <Nav.Bar.Center>
            <p>x</p>
          </Nav.Bar.Center>
        ),
        alignClass: "pluto--center",
      },
      {
        name: "End",
        element: (
          <Nav.Bar.End>
            <p>x</p>
          </Nav.Bar.End>
        ),
        alignClass: "pluto--end",
      },
      {
        name: "AbsoluteCenter",
        element: (
          <Nav.Bar.AbsoluteCenter>
            <p>x</p>
          </Nav.Bar.AbsoluteCenter>
        ),
        alignClass: "pluto--absolute-center",
      },
    ])(
      "should apply the alignment class for $name content",
      ({ element, alignClass }) => {
        const c = render(element);
        const content = contentOf(c);
        expect(content.className).toContain("pluto-navbar__content");
        expect(content.className).toContain(alignClass);
      },
    );

    it("should render plain content without an alignment modifier", () => {
      const c = render(
        <Nav.Bar.Content>
          <p>x</p>
        </Nav.Bar.Content>,
      );
      const content = contentOf(c);
      expect(content.className).toContain("pluto-navbar__content");
      expect(content.className).not.toContain("pluto--start");
      expect(content.className).not.toContain("pluto--end");
      expect(content.className).not.toContain("pluto--center");
    });

    it("should add a border on the aligned side for bordered content", () => {
      const c = render(
        <Nav.Bar.Start bordered>
          <p>x</p>
        </Nav.Bar.Start>,
      );
      expect(contentOf(c).className).toContain("pluto--bordered-start");
    });

    it("should not add a border to plain bordered content", () => {
      const c = render(
        <Nav.Bar.Content bordered>
          <p>x</p>
        </Nav.Bar.Content>,
      );
      expect(contentOf(c).className).not.toContain("pluto--bordered");
    });

    it("should forward a class name to content", () => {
      const c = render(
        <Nav.Bar.Start className="custom-content">
          <p>x</p>
        </Nav.Bar.Start>,
      );
      expect(contentOf(c).className).toContain("custom-content");
    });

    it("should render multiple content sections within a bar", () => {
      const c = render(
        <Nav.Bar location="left">
          <Nav.Bar.Start>
            <p>Start</p>
          </Nav.Bar.Start>
          <Nav.Bar.End>
            <p>End</p>
          </Nav.Bar.End>
        </Nav.Bar>,
      );
      expect(c.getByText("Start")).toBeTruthy();
      expect(c.getByText("End")).toBeTruthy();
    });
  });
});
