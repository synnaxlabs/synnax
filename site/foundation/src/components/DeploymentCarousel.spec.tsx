// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { type ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DeploymentCarousel } from "@/components/DeploymentCarousel";
import { loadScenes } from "@/components/deployments/loadScenes";

// Isolate async loading from the SVG geometry, so navigation, rejected imports,
// and stale requests can be tested without coupling to a scene's visual details.
vi.mock("@/components/deployments/loadScenes", () => ({
  loadScenes: {
    industrial: vi.fn(),
    aerospace: vi.fn(),
    quantum: vi.fn(),
    energy: vi.fn(),
    marine: vi.fn(),
  },
}));
vi.mock("@/components/deployments/DeploymentGeometry", () => ({
  Datum: () => <g />,
}));

const DEPLOYMENTS = [
  {
    id: "industrial",
    label: "Process plants",
    title: "The whole plant, in the same context.",
  },
  {
    id: "aerospace",
    label: "Aerospace",
    title: "From the test bench to the launch pad.",
  },
  {
    id: "quantum",
    label: "Quantum labs",
    title: "A connected cryogenic facility.",
  },
  {
    id: "energy",
    label: "Energy storage",
    title: "Many sites. One operating picture.",
  },
  {
    id: "marine",
    label: "Marine fleets",
    title: "On every vessel. Across the fleet.",
  },
];

const expectSelection = (index: number): HTMLButtonElement => {
  const deployment = DEPLOYMENTS[index];
  const selected = screen.getByRole<HTMLButtonElement>("tab", {
    name: deployment.label,
    selected: true,
  });
  const panel = screen.getByRole("tabpanel", { name: deployment.label });
  expect(screen.getAllByRole("tabpanel")).toHaveLength(1);
  expect(panel.getAttribute("aria-labelledby")).toBe(selected.id);
  expect(selected.getAttribute("aria-controls")).toBe(panel.id);
  expect(screen.getAllByRole("tab", { selected: true })).toEqual([selected]);
  for (const tab of screen.getAllByRole("tab"))
    expect(tab.tabIndex).toBe(tab === selected ? 0 : -1);

  const scenes = screen.getAllByTestId("deployment-scene");
  expect(scenes).toHaveLength(1);
  expect(scenes[0].getAttribute("data-deployment")).toBe(deployment.id);
  const images = screen.getAllByRole("img");
  expect(images).toHaveLength(1);
  expect(images[0].querySelector("title")?.textContent).toBe(deployment.title);
  expect(screen.getByRole("heading", { name: deployment.title })).toBeTruthy();
  return selected;
};

describe("DeploymentCarousel", () => {
  beforeEach(() => {
    for (const id of Object.keys(loadScenes) as (keyof typeof loadScenes)[]) {
      const Scene = (): ReactElement => (
        <g data-testid="deployment-scene" data-deployment={id} />
      );
      vi.mocked(loadScenes[id]).mockReset().mockResolvedValue(Scene);
    }
  });
  afterEach(cleanup);

  it("should server-render the carousel shell without loading any scene", () => {
    const html = renderToStaticMarkup(<DeploymentCarousel />);
    expect(html).toContain("The whole plant, in the same context.");
    expect(html).not.toContain('data-testid="deployment-scene"');
    for (const load of Object.values(loadScenes)) expect(load).not.toHaveBeenCalled();
  });

  it("should retain the newest selection when an earlier scene finishes later", async () => {
    let resolvePrevious!: (scene: () => ReactElement) => void;
    vi.mocked(loadScenes.aerospace).mockReturnValue(
      new Promise((resolve) => {
        resolvePrevious = resolve;
      }),
    );
    render(<DeploymentCarousel />);
    await waitFor(() => expectSelection(0));
    fireEvent.click(screen.getByRole("tab", { name: "Aerospace" }));
    expect(screen.getByRole("tabpanel").getAttribute("aria-busy")).toBe("true");
    fireEvent.click(screen.getByRole("tab", { name: "Quantum labs" }));
    await waitFor(() => expectSelection(2));
    await act(async () => {
      resolvePrevious(() => (
        <g data-testid="deployment-scene" data-deployment="aerospace" />
      ));
    });
    expectSelection(2);
  });

  it("should warm only adjacent scenes after the visible carousel becomes idle", async () => {
    let intersection!: IntersectionObserverCallback;
    const idle = vi.fn<(callback: IdleRequestCallback) => number>().mockReturnValue(1);
    vi.stubGlobal("requestIdleCallback", idle);
    vi.stubGlobal("cancelIdleCallback", vi.fn());
    vi.stubGlobal(
      "IntersectionObserver",
      class {
        constructor(callback: IntersectionObserverCallback) {
          intersection = callback;
        }
        observe = vi.fn();
        disconnect = vi.fn();
      },
    );
    render(<DeploymentCarousel />);
    await waitFor(() => expectSelection(0));
    expect(idle).not.toHaveBeenCalled();
    act(() =>
      intersection(
        [{ isIntersecting: true } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      ),
    );
    await waitFor(() => expect(idle).toHaveBeenCalledOnce());
    act(() => idle.mock.calls[0][0]({ didTimeout: false, timeRemaining: () => 30 }));
    expect(loadScenes.aerospace).toHaveBeenCalledOnce();
    expect(loadScenes.marine).toHaveBeenCalledOnce();
    expect(loadScenes.quantum).not.toHaveBeenCalled();
    expect(loadScenes.energy).not.toHaveBeenCalled();
    expectSelection(0);
  });

  it("should let a failed scene request be retried", async () => {
    vi.mocked(loadScenes.industrial).mockRejectedValueOnce(new Error("Offline"));
    render(<DeploymentCarousel />);
    const retry = await screen.findByRole("button", {
      name: "Retry loading deployment",
    });
    expect(screen.getByRole("tabpanel").getAttribute("aria-busy")).toBe("false");
    fireEvent.click(retry);
    await waitFor(() => expectSelection(0));
    expect(
      screen.queryByRole("button", { name: "Retry loading deployment" }),
    ).toBeNull();
  });

  it("should expose only the selected scene and synchronize the tabpanel label", async () => {
    render(<DeploymentCarousel />);
    await waitFor(() => expectSelection(0));
    expect(
      screen.getByRole("tablist", { name: "Infrastructure deployments" }),
    ).toBeTruthy();
    expect(screen.getAllByRole("tab")).toHaveLength(5);

    for (const index of [3, 1, 4, 2, 0]) {
      fireEvent.click(screen.getByRole("tab", { name: DEPLOYMENTS[index].label }));
      await waitFor(() => expectSelection(index));
    }
  });

  it.each([
    { control: "Next deployment", order: [1, 2, 3, 4, 0] },
    { control: "Previous deployment", order: [4, 3, 2, 1, 0] },
  ])(
    "should wrap through all five scenes with $control",
    async ({ control, order }) => {
      render(<DeploymentCarousel />);
      const button = screen.getByRole("button", { name: control });
      button.focus();
      for (const index of order) {
        fireEvent.click(button);
        await waitFor(() => expectSelection(index));
        expect(document.activeElement).toBe(button);
      }
    },
  );

  it("should move selection and focus with arrow keys, Home, and End", async () => {
    render(<DeploymentCarousel />);
    let selected = await waitFor(() => expectSelection(0));
    selected.focus();

    for (const [key, index] of [
      ["ArrowLeft", 4],
      ["ArrowRight", 0],
      ["ArrowRight", 1],
      ["End", 4],
      ["Home", 0],
      ["ArrowRight", 1],
      ["ArrowLeft", 0],
    ] as const) {
      fireEvent.keyDown(selected, { key });
      selected = await waitFor(() => expectSelection(index));
      expect(document.activeElement).toBe(selected);
    }
  });
});
