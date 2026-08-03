// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

// AuxTop only renders the controls toggle on macOS/Windows, so pin the OS to keep the
// controls assertions deterministic across host platforms (Linux CI included).
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

describe("app/nav/bar/AuxTop", () => {
  describe("title", () => {
    it("should render the active project name", async () => {
      await renderBar(<Bar.AuxTop />, withActiveProject());
      expect(await screen.findByText("Ops", {})).toBeDefined();
    });
  });

  describe("controls", () => {
    it("should render the controls toggle button", async () => {
      await renderBar(<Bar.AuxTop />, withActiveProject());
      expect(await screen.findByText("Controls", {})).toBeDefined();
    });

    it("should toggle the bottom drawer open when clicked", async () => {
      const { store } = await renderBar(<Bar.AuxTop />, withActiveProject());
      fireEvent.click(await screen.findByText("Controls", {}));
      await waitFor(() => expect(bottom(store).visible).toBe(true));
      expect(bottom(store).hover).toBe(true);
    });

    it("should drop the hover on a second toggle, keeping the drawer pinned", async () => {
      const { store } = await renderBar(<Bar.AuxTop />, withActiveProject());
      const button = await screen.findByText("Controls", {});
      fireEvent.click(button);
      fireEvent.click(button);
      await waitFor(() => expect(bottom(store).hover).toBe(false));
      expect(bottom(store).visible).toBe(true);
    });
  });
});
