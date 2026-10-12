// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { type SystemLayer } from "@/components/IndustrialScene";
import { SystemDemo } from "@/components/SystemDemo";

// These tests exercise the controls and state transitions independently of the scene.
vi.mock("@/components/IndustrialScene", () => ({
  IndustrialScene: ({ layer }: { layer: SystemLayer }) => (
    <div data-testid="industrial-scene" data-layer={layer} />
  ),
}));

const steps = [
  {
    label: "Equipment",
    layer: "operation",
    title: "Start with the real world.",
    detail:
      "Foundation nodes stream data and carry supervisory commands to and from your existing systems.",
  },
  {
    label: "Cluster",
    layer: "connectivity",
    title: "Form the cluster.",
    detail:
      "Foundation nodes communicate within and across sites, bringing these systems into one cluster.",
  },
  {
    label: "Telemetry",
    layer: "applications",
    title: "Build a real-time picture.",
    detail:
      "Give every application and agent the same live context from your operation.",
  },
  {
    label: "Control",
    layer: "control",
    title: "Act on the physical world.",
    detail:
      "Foundation routes supervisory commands to any piece of your infrastructure. It protects critical systems through sophisticated rules, interlocks, and access control mechanisms.",
  },
];

const expectStep = (index: number): void => {
  const step = steps[index];
  const explanation = screen.getByRole("article", { name: step.title });
  expect(explanation.querySelectorAll("p")).toHaveLength(2);
  expect(screen.getByText(step.detail)).toBeTruthy();
  expect(screen.getByTestId("industrial-scene").getAttribute("data-layer")).toBe(
    step.layer,
  );
  expect(screen.getByText(`0${index + 1} of 04 · ${step.label}`)).toBeTruthy();
  expect(screen.getAllByRole("button", { pressed: true })).toHaveLength(1);
  steps.forEach(({ label, title }, i) => {
    const button = screen.getByRole("button", { name: new RegExp(label) });
    expect(button.getAttribute("aria-pressed")).toBe(String(index === i));
    expect(button.getAttribute("aria-controls")).toBe(explanation.id);
    if (i !== index) expect(screen.queryByRole("heading", { name: title })).toBeNull();
  });
};

describe("SystemDemo", () => {
  afterEach(cleanup);

  it("should keep the selected stage, explanation, and scene in sync", async () => {
    const { container } = render(<SystemDemo />);
    await screen.findByTestId("industrial-scene");
    const diagram = container.querySelector(".system-demo")!;
    expectStep(0);

    for (const index of [3, 1, 2, 0]) {
      fireEvent.click(
        screen.getByRole("button", { name: new RegExp(steps[index].label) }),
      );
      expectStep(index);
      expect(diagram.getAttribute("data-layer")).toBe(steps[index].layer);
    }
  });

  it("should advance through all four stages and return to the start", async () => {
    render(<SystemDemo />);
    await screen.findByTestId("industrial-scene");
    const next = screen.getByRole("button", { name: "Next step" });
    expect(next.getAttribute("type")).toBe("button");
    next.focus();
    for (const index of [1, 2, 3, 0]) {
      fireEvent.click(next);
      expectStep(index);
      expect(document.activeElement).toBe(next);
    }
  });

  it("should announce the active explanation and retain native keyboard controls", () => {
    render(<SystemDemo />);
    const explanation = screen.getByRole("article", {
      name: "Start with the real world.",
    });
    const liveCopy = explanation.querySelector('[aria-live="polite"]');
    expect(liveCopy?.getAttribute("aria-atomic")).toBe("true");
    const stageGroup = screen.getByRole("group", { name: "How Foundation works" });
    for (const button of stageGroup.querySelectorAll("button")) {
      expect(button.type).toBe("button");
      expect(button.disabled).toBe(false);
      expect(button.tabIndex).toBe(0);
    }
    fireEvent.click(screen.getByRole("button", { name: /Control/ }));
    expect(liveCopy?.textContent).toContain("Act on the physical world.");
  });
});
