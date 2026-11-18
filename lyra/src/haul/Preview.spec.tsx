// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { describe, expect, it, onTestFinished } from "vitest";

import { Haul } from "@/haul";

describe("Haul.Preview", () => {
  it("should show the name, then the facts between separators", () => {
    const { container } = render(
      <Haul.Preview name="pressure" facts={["float64", "Index"]} />,
    );
    expect(screen.getByText("pressure").tagName).toBe("P");
    const facts = container.querySelector(".lyra-haul-preview__facts");
    expect(facts?.textContent).toBe("float64·Index");
  });

  it("should leave out the facts line when there are no facts", () => {
    const { container } = render(<Haul.Preview name="pressure" />);
    expect(container.querySelector(".lyra-haul-preview__facts")).toBeNull();
  });

  it("should put the icon in the tile", () => {
    const { container } = render(
      <Haul.Preview icon={<svg aria-label="channel icon" />} name="pressure" />,
    );
    const tile = container.querySelector(".lyra-haul-preview__tile");
    expect(tile?.contains(screen.getByLabelText("channel icon"))).toBe(true);
  });

  it("should fill the tile with the swatch in place of the icon", () => {
    const { container } = render(
      <Haul.Preview
        icon={<svg aria-label="range icon" />}
        swatch="#ff0000"
        name="Run 12"
      />,
    );
    const tile = container.querySelector<HTMLElement>(".lyra-haul-preview__tile");
    expect(tile?.style.background).toBe("rgb(255, 0, 0)");
    expect(screen.queryByLabelText("range icon")).toBeNull();
  });

  it("should leave out the tile when there is no icon or swatch", () => {
    const { container } = render(<Haul.Preview name="pressure" />);
    expect(container.querySelector(".lyra-haul-preview__tile")).toBeNull();
  });

  it("should stack with a count when the drag carries many objects", () => {
    const { container } = render(<Haul.Preview name="pressure" count={3} />);
    expect(container.firstElementChild?.classList).toContain(
      "lyra-haul-preview--stack",
    );
    expect(container.querySelector(".lyra-haul-preview__count")?.textContent).toBe("3");
  });

  it("should show no stack or count for one object", () => {
    const { container } = render(<Haul.Preview name="pressure" count={1} />);
    expect(container.firstElementChild?.classList).not.toContain(
      "lyra-haul-preview--stack",
    );
    expect(container.querySelector(".lyra-haul-preview__count")).toBeNull();
  });
});

describe("Haul.Copy", () => {
  const renderSource = (): HTMLElement => {
    const strip = document.createElement("div");
    strip.className = "tabs";
    strip.innerHTML =
      '<div class="tab" style="margin: 4px"><svg></svg>Plot' +
      `<button ${Haul.OMIT_ATTRIBUTE}>x</button></div>`;
    document.body.append(strip);
    onTestFinished(() => strip.remove());
    return strip.querySelector<HTMLElement>(".tab")!;
  };

  it("should copy the element without the parts marked to leave out", () => {
    const source = renderSource();
    const { container } = render(<Haul.Copy of={source} />);
    const copy = container.querySelector<HTMLElement>(".tab");
    expect(copy?.textContent).toBe("Plot");
    expect(copy?.querySelector("svg")).not.toBeNull();
    expect(source.querySelector("button")).not.toBeNull();
  });

  it("should drop the margin, transform, and opacity of the copied element", () => {
    const source = renderSource();
    Object.assign(source.style, { transform: "scale(0.9)", opacity: "0.5" });
    const { container } = render(<Haul.Copy of={source} />);
    const copy = container.querySelector<HTMLElement>(".tab");
    expect(copy?.style.margin).toBe("0px");
    expect(copy?.style.transform).toBe("none");
    expect(copy?.style.opacity).toBe("1");
  });

  it("should wrap the copy in stand-ins that keep the classes of its ancestors", () => {
    const { container } = render(<Haul.Copy of={renderSource()} />);
    const copy = container.querySelector(".tab");
    const standIn = copy?.closest<HTMLElement>(".tabs");
    expect(standIn?.style.display).toBe("contents");
    expect(container.contains(standIn ?? null)).toBe(true);
  });
});
