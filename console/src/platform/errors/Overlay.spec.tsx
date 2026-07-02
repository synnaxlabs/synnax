// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({
  engine: "web",
}));

vi.mock("@/session/runtime/runtime", () => ({
  get ENGINE() {
    return mocks.engine;
  },
  Drift: class {},
}));

const windowControls = {
  show: vi.fn().mockResolvedValue(undefined),
  close: vi.fn().mockResolvedValue(undefined),
  minimize: vi.fn().mockResolvedValue(undefined),
  maximize: vi.fn().mockResolvedValue(undefined),
};

vi.mock("@tauri-apps/api/window", () => ({
  getCurrentWindow: () => windowControls,
}));

vi.mock("@tauri-apps/api/app", () => ({
  getVersion: vi.fn().mockResolvedValue("9.9.9"),
}));

import { getVersion } from "@tauri-apps/api/app";

import { Errors } from "@/platform/errors";
import { Session } from "@/session";
import { createTestStore, renderWithConsole } from "@/testutil";

const BOOM = "kaboom detonated";

const Boom = (): ReactElement => {
  throw new Error(BOOM);
};

describe("Errors.Overlay", () => {
  let consoleError: ReturnType<typeof vi.spyOn>;
  beforeEach(() => {
    mocks.engine = "web";
    vi.mocked(getVersion).mockClear();
    for (const c of Object.values(windowControls)) c.mockClear();
    consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
  });
  afterEach(() => {
    consoleError.mockRestore();
  });

  describe("OverlayWithStore", () => {
    it("should render its children when nothing throws", async () => {
      await renderWithConsole(
        <Errors.OverlayWithStore>
          <span>all good</span>
        </Errors.OverlayWithStore>,
      );
      expect(screen.getByText("all good")).toBeTruthy();
    });

    it("should render the fallback with the thrown error's details", async () => {
      await renderWithConsole(
        <Errors.OverlayWithStore>
          <Boom />
        </Errors.OverlayWithStore>,
      );
      expect(await screen.findByText(BOOM)).toBeTruthy();
      expect(screen.getByText("Something went wrong")).toBeTruthy();
    });

    it("should dispatch revertState when Reload Console is clicked", async () => {
      const store = await createTestStore();
      const dispatch = vi.spyOn(store, "dispatch");
      await renderWithConsole(
        <Errors.OverlayWithStore>
          <Boom />
        </Errors.OverlayWithStore>,
        { store },
      );
      const button = await screen.findByRole("button", { name: "Reload Console" });
      fireEvent.click(button);
      await waitFor(() =>
        expect(dispatch).toHaveBeenCalledWith(Session.Persist.revertState()),
      );
    });

    it("should dispatch clearState when the clear button is clicked", async () => {
      const store = await createTestStore();
      const dispatch = vi.spyOn(store, "dispatch");
      await renderWithConsole(
        <Errors.OverlayWithStore>
          <Boom />
        </Errors.OverlayWithStore>,
        { store },
      );
      const button = await screen.findByRole("button", {
        name: /Clear storage and reload Console/,
      });
      fireEvent.click(button);
      await waitFor(() =>
        expect(dispatch).toHaveBeenCalledWith(Session.Persist.clearState()),
      );
    });

    it("should route the window controls to the tauri window under tauri", async () => {
      mocks.engine = "tauri";
      const { container } = await renderWithConsole(
        <Errors.OverlayWithStore>
          <Boom />
        </Errors.OverlayWithStore>,
      );
      await screen.findByText(BOOM);
      const click = (suffix: string) => {
        const el = container.querySelector(`.pluto-windows-control--${suffix}`);
        expect(el).not.toBeNull();
        fireEvent.click(el!);
      };
      click("close");
      click("minimize");
      click("maximize");
      await waitFor(() => expect(windowControls.close).toHaveBeenCalled());
      expect(windowControls.minimize).toHaveBeenCalled();
      expect(windowControls.maximize).toHaveBeenCalled();
    });
  });

  describe("OverlayWithoutStore", () => {
    it("should render its children when nothing throws", async () => {
      await renderWithConsole(
        <Errors.OverlayWithoutStore>
          <span>steady state</span>
        </Errors.OverlayWithoutStore>,
      );
      expect(screen.getByText("steady state")).toBeTruthy();
    });

    it("should render the fallback and omit the retry button without a store", async () => {
      await renderWithConsole(
        <Errors.OverlayWithoutStore>
          <Boom />
        </Errors.OverlayWithoutStore>,
      );
      expect(await screen.findByText(BOOM)).toBeTruthy();
      expect(
        screen.getByRole("button", { name: /Clear storage and reload Console/ }),
      ).toBeTruthy();
      expect(screen.queryByRole("button", { name: "Reload Console" })).toBeNull();
    });

    it("should show the window and load the console version under tauri", async () => {
      mocks.engine = "tauri";
      await renderWithConsole(
        <Errors.OverlayWithoutStore>
          <Boom />
        </Errors.OverlayWithoutStore>,
      );
      expect(await screen.findByText(BOOM)).toBeTruthy();
      await waitFor(() => expect(windowControls.show).toHaveBeenCalled());
      expect(getVersion).toHaveBeenCalled();
    });
  });
});
