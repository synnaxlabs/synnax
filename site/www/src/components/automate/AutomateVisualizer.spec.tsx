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

import { AutomateVisualizer } from "@/components/automate/AutomateVisualizer";
import { EXAMPLES } from "@/components/automate/timeline";

const ERROR = "var(--pluto-error-z)";
const DIM = "var(--pluto-gray-l5)";

const CODE = EXAMPLES.map((_, i) =>
  Array.from(
    { length: 24 },
    (_, l) => `<span class="line">ex${i}-l${l + 1}</span>`,
  ).join("\n"),
);

const advance = (ms: number): void =>
  act(() => {
    vi.advanceTimersByTime(ms);
  });

// Plays the active example from the start of one step to the start of another. Each
// step needs its own advance, as React schedules the next step after a render.
const play = (example: number, from: number, to: number): void => {
  for (const { duration } of EXAMPLES[example].steps.slice(from, to)) advance(duration);
};

// The node label renders before its value.
const iconColor = (label: string): string =>
  screen.getAllByText(label)[0].closest("g")!.querySelector<SVGElement>("div svg")!
    .style.color;

describe("AutomateVisualizer", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("pressurization", () => {
    it("should fill the tank as the sequence runs", () => {
      render(<AutomateVisualizer codeHtmls={CODE} />);
      expect(screen.getByText("0.0 psi")).toBeTruthy();
      play(0, 0, 3);
      expect(screen.getByText("520.0 psi")).toBeTruthy();
      expect(screen.getByText("press")).toBeTruthy();
    });

    it("should hide the authority meter", () => {
      const { container } = render(<AutomateVisualizer codeHtmls={CODE} />);
      expect(container.textContent).not.toContain("auth");
    });
  });

  describe("abort", () => {
    it("should show the authority rising to 255 in the emergency", () => {
      const { container } = render(<AutomateVisualizer codeHtmls={CODE} />);
      fireEvent.click(screen.getByText("Abort sequence"));
      expect(container.textContent).toContain("auth 200");
      play(2, 0, 6);
      expect(container.textContent).toContain("auth 255");
      expect(screen.getByText("emergency")).toBeTruthy();
    });

    it("should start a fresh tank when the tab changes", () => {
      render(<AutomateVisualizer codeHtmls={CODE} />);
      play(0, 0, 3);
      fireEvent.click(screen.getByText("Abort sequence"));
      expect(screen.getByText("0.0 psi")).toBeTruthy();
    });
  });

  describe("alarm", () => {
    const open = (): void => {
      render(<AutomateVisualizer codeHtmls={CODE} />);
      fireEvent.click(screen.getByText("Alarm monitoring"));
    };

    it("should show the flow diagram in place of the tank", () => {
      open();
      expect(screen.queryByText(/psi$/)).toBeNull();
      expect(screen.getByText("400 PSI")).toBeTruthy();
    });

    it("should report a reading under the limit as nominal", () => {
      open();
      play(1, 0, 2);
      expect(screen.getByText("< 750")).toBeTruthy();
      play(1, 2, 4);
      expect(screen.getAllByText("nominal")).toHaveLength(2);
      expect(iconColor("warning")).toBe(DIM);
    });

    it("should raise the warning in red for a reading over the limit", () => {
      open();
      play(1, 0, 6);
      expect(screen.getByText("760 PSI")).toBeTruthy();
      expect(screen.getByText("> 750")).toBeTruthy();
      play(1, 6, 8);
      expect(screen.getAllByText("warning")).toHaveLength(2);
      expect(iconColor("warning")).toBe(ERROR);
    });

    it("should show the alarm code", () => {
      const { container } = render(<AutomateVisualizer codeHtmls={CODE} />);
      fireEvent.click(screen.getByText("Alarm monitoring"));
      const active = [...container.querySelectorAll('.line[data-active="true"]')];
      expect(active.map((l) => l.textContent)).toEqual(["ex1-l1", "ex1-l2", "ex1-l3"]);
    });
  });
});
