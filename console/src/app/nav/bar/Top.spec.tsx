// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

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

    it("should toggle the bottom drawer open when clicked", async () => {
      const { store } = await renderBar(<Bar.Top secondary />, withActiveProject());
      fireEvent.click(await screen.findByText("Controls", {}));
      await waitFor(() => expect(bottom(store).visible).toBe(true));
      expect(bottom(store).hover).toBe(true);
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

    it("should close the drawer on a second toggle", async () => {
      const { store } = await renderBar(<Bar.Top secondary />, withActiveProject());
      const button = await screen.findByText("Controls", {});
      fireEvent.click(button);
      fireEvent.click(button);
      await waitFor(() => expect(bottom(store).visible).toBe(false));
      expect(bottom(store).hover).toBe(false);
    });
  });
});
