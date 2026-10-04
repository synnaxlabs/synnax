// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CALC_EXAMPLES } from "@/components/stream/calcTimeline";
import { CalcVisualizer } from "@/components/stream/CalcVisualizer";

const code = (example: number): string =>
  Array.from(
    { length: 12 },
    (_, i) => `<span class="line">ex${example}-l${i + 1}</span>`,
  ).join("\n");

const CODE = CALC_EXAMPLES.map((_, i) => code(i));

const highlighted = (container: HTMLElement): string[] =>
  [...container.querySelectorAll('.line[data-active="true"]')].map(
    (el) => el.textContent ?? "",
  );

const dots = (container: HTMLElement): HTMLElement[] => [
  ...container.querySelectorAll<HTMLElement>(".viz-dot"),
];

const advance = (ms: number): void =>
  act(() => {
    vi.advanceTimersByTime(ms);
  });

describe("CalcVisualizer", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should open on the first example", () => {
    const { container } = render(<CalcVisualizer codeHtmls={CODE} />);
    expect(screen.getByText("Conversion").className).toContain("viz-tab--active");
    expect(highlighted(container)).toEqual(["ex0-l1"]);
    expect(screen.getByText("22.3 °C")).toBeTruthy();
  });

  it("should step the diagram through the example", () => {
    render(<CalcVisualizer codeHtmls={CODE} />);
    expect(screen.queryByText("72.1 °F")).toBeNull();
    advance(1200);
    advance(1400);
    expect(screen.getByText("72.1 °F")).toBeTruthy();
  });

  it("should show the code and diagram of the selected example", () => {
    const { container } = render(<CalcVisualizer codeHtmls={CODE} />);
    fireEvent.click(screen.getByText("Sensor voting"));
    expect(highlighted(container)).toEqual(["ex4-l1", "ex4-l2", "ex4-l3"]);
    expect(screen.getByText("748.2")).toBeTruthy();
    expect(screen.queryByText("22.3 °C")).toBeNull();
  });

  it("should follow the highlighted lines of each step", () => {
    const { container } = render(<CalcVisualizer codeHtmls={CODE} />);
    fireEvent.click(screen.getByText("Sensor voting"));
    advance(1400);
    expect(highlighted(container)).toEqual(["ex4-l5", "ex4-l6"]);
  });

  it("should show one progress dot per step", () => {
    const { container } = render(<CalcVisualizer codeHtmls={CODE} />);
    expect(dots(container)).toHaveLength(CALC_EXAMPLES[0].steps.length);
    fireEvent.click(screen.getByText("Sensor voting"));
    expect(dots(container)).toHaveLength(CALC_EXAMPLES[4].steps.length);
  });

  it("should mark the dot of the current step with its duration", () => {
    const { container } = render(<CalcVisualizer codeHtmls={CODE} />);
    advance(1200);
    const active = dots(container).filter((d) =>
      d.classList.contains("viz-dot--active"),
    );
    expect(active).toEqual([dots(container)[1]]);
    expect(active[0].style.getPropertyValue("--step-duration")).toBe("1400ms");
  });

  it("should pause while the pointer is over it", () => {
    const { container } = render(<CalcVisualizer codeHtmls={CODE} />);
    fireEvent.mouseEnter(container.querySelector(".calc-visualizer")!);
    advance(10000);
    expect(screen.queryByText("72.1 °F")).toBeNull();
  });
});
