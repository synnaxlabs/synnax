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
  within,
} from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { EngineeringDemo } from "@/components/EngineeringDemo";

const signal = (name: string): HTMLElement =>
  screen.getByText(name, { selector: "dt" }).parentElement!;

describe("EngineeringDemo", () => {
  beforeAll(async () => {
    await import("@/components/EngineeringScene");
  });
  afterEach(cleanup);

  it("distinguishes source silence from connection health and preserves the last sample quality", async () => {
    render(<EngineeringDemo />);
    await screen.findByRole("img");
    fireEvent.click(screen.getByRole("button", { name: "Source silent" }));
    expect(
      screen.getByRole("img", {
        name: /Telemetry health, source silent.*connection stays online/i,
      }),
    ).toBeTruthy();
    expect(
      within(signal("Source quality")).getByText("Good · last sample"),
    ).toBeTruthy();
    expect(within(signal("Validation")).getByText("Stale · 18 s old")).toBeTruthy();
    expect(within(signal("Delivery")).getByText("Last sample confirmed")).toBeTruthy();
    expect(screen.getByText(/Expected every 1 s/)).toBeTruthy();
  });

  it("keeps fresh source data distinct from destination lag and reports retained history and explicit recovery gaps", async () => {
    render(<EngineeringDemo />);
    await screen.findByRole("img");
    fireEvent.click(screen.getByRole("button", { name: "Destination delayed" }));
    expect(within(signal("Source quality")).getByText("Good")).toBeTruthy();
    expect(within(signal("Validation")).getByText("Fresh · 180 ms old")).toBeTruthy();
    expect(within(signal("Delivery")).getByText("Delayed · 42 s lag")).toBeTruthy();
    expect(screen.getByText("18,400 samples pending")).toBeTruthy();
    expect(screen.getByText("Buffered · no confirmed loss")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Recovering" }));
    expect(
      screen.getByRole("img", {
        name: /Telemetry health, recovering.*four-second interval exceeded retention/i,
      }),
    ).toBeTruthy();
    expect(screen.getByText("4,200 samples pending")).toBeTruthy();
    expect(screen.getByText("4 s gap · outside retention")).toBeTruthy();
    expect(screen.getByText("Expired commands are never replayed")).toBeTruthy();
    expect(screen.queryByText("Buffered · no confirmed loss")).toBeNull();
    expect(
      screen.getByRole("button", { name: "Recovering", pressed: true }),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Destination delayed", pressed: false }),
    ).toBeTruthy();
  });

  it("exposes a value check without replacing source quality or claiming delivery failure", async () => {
    render(<EngineeringDemo />);
    await screen.findByRole("img");
    fireEvent.click(screen.getByRole("button", { name: "Value check" }));
    expect(within(signal("Source quality")).getByText("Good")).toBeTruthy();
    expect(
      within(signal("Validation")).getByText("86.4 bar · out of range"),
    ).toBeTruthy();
    expect(within(signal("Delivery")).getByText("Confirmed · 210 ms lag")).toBeTruthy();
    expect(screen.getByText("Validation does not rewrite source quality")).toBeTruthy();
  });

  it("resets the scenario when changing attributes, including from a reliability-only scenario", async () => {
    render(<EngineeringDemo />);
    await screen.findByRole("img");
    const attributes = within(
      screen.getByRole("group", { name: "Engineering attributes" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Value check" }));
    fireEvent.click(attributes.getByRole("button", { name: "Performance" }));
    const performance = within(
      screen.getByRole("group", { name: "performance scenarios" }),
    );
    expect(performance.getAllByRole("button", { pressed: true })).toEqual([
      performance.getByRole("button", { name: "Ingest" }),
    ]);
    expect(screen.getByRole("img", { name: /^The data path, step 1/ })).toBeTruthy();
    expect(screen.queryByText("Source quality", { selector: "dt" })).toBeNull();

    fireEvent.click(performance.getByRole("button", { name: "Deliver" }));
    fireEvent.click(attributes.getByRole("button", { name: "Deployment" }));
    expect(
      screen.getByRole("button", { name: "One node", pressed: true }),
    ).toBeTruthy();
    expect(
      screen.getByRole("img", { name: /^Deployment topology, step 1/ }),
    ).toBeTruthy();
    fireEvent.click(attributes.getByRole("button", { name: "Reliability" }));
    expect(screen.getByRole("button", { name: "Flowing", pressed: true })).toBeTruthy();
    expect(within(signal("Validation")).getByText("Fresh · 180 ms old")).toBeTruthy();
  });

  it("pauses decorative flow offscreen, in hidden tabs, and for reduced motion while scenarios remain operable", async () => {
    let intersect: IntersectionObserverCallback | undefined;
    let reduced = false;
    let preferenceChanged: (() => void) | undefined;
    const disconnect = vi.fn();
    vi.stubGlobal(
      "IntersectionObserver",
      class {
        observe = vi.fn();
        disconnect = disconnect;
        constructor(callback: IntersectionObserverCallback) {
          intersect = callback;
        }
      },
    );
    vi.stubGlobal("matchMedia", () => ({
      get matches() {
        return reduced;
      },
      addEventListener: (_name: string, callback: () => void) => {
        preferenceChanged = callback;
      },
      removeEventListener: vi.fn(),
    }));
    const hidden = vi.spyOn(document, "hidden", "get").mockReturnValue(false);
    const { container, unmount } = render(<EngineeringDemo />);
    await screen.findByRole("img");
    const demo = container.querySelector<HTMLElement>(".engineering-demo")!;
    expect(demo.dataset.motion).toBe("paused");
    act(() =>
      intersect?.(
        [{ isIntersecting: true } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      ),
    );
    expect(demo.dataset.motion).toBe("running");
    hidden.mockReturnValue(true);
    fireEvent(document, new Event("visibilitychange"));
    expect(demo.dataset.motion).toBe("paused");
    hidden.mockReturnValue(false);
    fireEvent(document, new Event("visibilitychange"));
    expect(demo.dataset.motion).toBe("running");
    reduced = true;
    act(() => preferenceChanged?.());
    expect(demo.dataset.motion).toBe("paused");
    fireEvent.click(screen.getByRole("button", { name: "Source silent" }));
    expect(within(signal("Validation")).getByText("Stale · 18 s old")).toBeTruthy();
    unmount();
    expect(disconnect).toHaveBeenCalledOnce();
  });
});
