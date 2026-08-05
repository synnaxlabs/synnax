// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/pluto";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

// The controls toggle only renders on macOS/Windows, so pin the OS to keep its
// assertions deterministic across host platforms (Linux CI included).
await vi.hoisted(async () => {
  const { pinOS } = await import("@/testutil/pinOS");
  pinOS("macOS");
});

import { Bar } from "@/app/nav/bar";
import { renderBar, withActiveProject } from "@/app/nav/bar/testutil";
import { Session } from "@/session";
import { type TestStore } from "@/testutil";

const bottom = (store: TestStore) =>
  Session.Nav.selectWindowState(store.getState()).bottom;

describe("app/nav/bar/Top", () => {
  it("should render the active project name in the selector", async () => {
    await renderBar(<Bar.Top />, withActiveProject());
    expect(await screen.findByText("Ops", {})).toBeDefined();
  });

  it("should render the authenticated user from the live client", async () => {
    await renderBar(<Bar.Top />, withActiveProject());
    expect(await screen.findByText("synnax", {})).toBeDefined();
  });

  describe("secondary", () => {
    it("should not render the project selector", async () => {
      await renderBar(<Bar.Top secondary />, withActiveProject());
      expect(await screen.findByText("Controls", {})).toBeDefined();
      expect(screen.queryByText("Ops")).toBeNull();
    });

    it("should render the controls toggle button", async () => {
      await renderBar(<Bar.Top secondary />, withActiveProject());
      expect(await screen.findByText("Controls", {})).toBeDefined();
    });

    it("should open the drawer pinned when clicked", async () => {
      const { store } = await renderBar(<Bar.Top secondary />, withActiveProject());
      fireEvent.click(await screen.findByText("Controls", {}));
      await waitFor(() => expect(bottom(store).visible).toBe(true));
      expect(bottom(store).hover).toBe(false);
    });

    it("should toggle the bottom drawer on the V trigger", async () => {
      const { store } = await renderBar(
        <Triggers.Provider>
          <Bar.Top secondary />
        </Triggers.Provider>,
        withActiveProject(),
      );
      await screen.findByText("Controls", {});
      fireEvent.keyDown(document.body, { code: "KeyV" });
      fireEvent.keyUp(document.body, { code: "KeyV" });
      await waitFor(() => expect(bottom(store).visible).toBe(true));
    });

    it("should close the drawer on a second click", async () => {
      const { store } = await renderBar(<Bar.Top secondary />, withActiveProject());
      const button = await screen.findByText("Controls", {});
      fireEvent.click(button);
      fireEvent.click(button);
      await waitFor(() => expect(bottom(store).visible).toBe(false));
      expect(bottom(store).hover).toBe(false);
    });

    describe("hover", () => {
      afterEach(() => {
        vi.useRealTimers();
      });

      const findControls = async () =>
        await screen.findByRole("button", { name: /Controls/ });

      it("should preview the drawer after dwelling on the controls button", async () => {
        const { store } = await renderBar(<Bar.Top secondary />, withActiveProject());
        const button = await findControls();
        vi.useFakeTimers();
        fireEvent.mouseEnter(button, { clientX: 500, clientY: 10 });
        act(() => {
          vi.advanceTimersByTime(350);
        });
        expect(bottom(store).visible).toBe(true);
        expect(bottom(store).hover).toBe(true);
      });

      it("should dismiss the preview when the pointer drifts along the top bar", async () => {
        const { store } = await renderBar(<Bar.Top secondary />, withActiveProject());
        const button = await findControls();
        vi.useFakeTimers();
        fireEvent.mouseEnter(button, { clientX: 500, clientY: 10 });
        act(() => {
          vi.advanceTimersByTime(350);
        });
        act(() => {
          window.dispatchEvent(
            new MouseEvent("mousemove", { clientX: 700, clientY: 10 }),
          );
        });
        expect(bottom(store).hover).toBe(false);
        expect(bottom(store).visible).toBe(false);
      });

      it("should keep the preview open while the pointer moves toward the drawer", async () => {
        const { store } = await renderBar(<Bar.Top secondary />, withActiveProject());
        const button = await findControls();
        vi.useFakeTimers();
        fireEvent.mouseEnter(button, { clientX: 500, clientY: 10 });
        act(() => {
          vi.advanceTimersByTime(350);
        });
        act(() => {
          window.dispatchEvent(
            new MouseEvent("mousemove", { clientX: 500, clientY: 400 }),
          );
        });
        expect(bottom(store).hover).toBe(true);
        expect(bottom(store).visible).toBe(true);
      });

      it("should pin the drawer when clicked during a preview", async () => {
        const { store } = await renderBar(<Bar.Top secondary />, withActiveProject());
        const button = await findControls();
        vi.useFakeTimers();
        fireEvent.mouseEnter(button, { clientX: 500, clientY: 10 });
        act(() => {
          vi.advanceTimersByTime(350);
        });
        fireEvent.click(button);
        expect(bottom(store).visible).toBe(true);
        expect(bottom(store).hover).toBe(false);
      });
    });
  });
});
