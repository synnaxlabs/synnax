// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

// AuxTop only renders the controls toggle on macOS/Windows, so pin the detected OS to
// keep the controls assertions deterministic across host platforms (Linux CI included).
vi.mock("@synnaxlabs/pluto", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const OS = actual.OS as Record<string, unknown>;
  return { ...actual, OS: { ...OS, use: () => "macOS" } };
});

import { AuxTop } from "@/layered/app/nav/bar/AuxTop";
import { renderBar, TIMEOUT, withActiveProject } from "@/layered/app/nav/bar/testutil";
import { Session } from "@/layered/session";
import { Layout } from "@/layout";

const TAB = "plot-1";

const withActiveTab = (name: string) =>
  withActiveProject({
    [Layout.SLICE_NAME]: {
      ...Layout.ZERO_SLICE_STATE,
      layouts: {
        ...Layout.ZERO_SLICE_STATE.layouts,
        [TAB]: {
          key: TAB,
          name,
          windowKey: MAIN_WINDOW,
          type: "lineplot",
          location: "mosaic",
        },
      },
      mosaics: {
        ...Layout.ZERO_SLICE_STATE.mosaics,
        [MAIN_WINDOW]: {
          ...Layout.ZERO_SLICE_STATE.mosaics[MAIN_WINDOW],
          activeTab: TAB,
        },
      },
    },
  });

const bottom = (store: { getState: () => unknown }) =>
  Session.Nav.selectWindowState(store.getState() as never).bottom;

describe("app/nav/bar/AuxTop", () => {
  describe("title", () => {
    it("should render the active project name", async () => {
      await renderBar(<AuxTop />, withActiveProject());
      expect(await screen.findByText(/- Ops/, {}, TIMEOUT)).toBeDefined();
    });

    it("should render the active mosaic tab name alongside the project", async () => {
      await renderBar(<AuxTop />, withActiveTab("My Plot"));
      expect(await screen.findByText(/My Plot - Ops/, {}, TIMEOUT)).toBeDefined();
    });
  });

  describe("controls", () => {
    it("should render the controls toggle button", async () => {
      await renderBar(<AuxTop />, withActiveProject());
      expect(await screen.findByText("Controls", {}, TIMEOUT)).toBeDefined();
    });

    it("should toggle the bottom drawer open when clicked", async () => {
      const { store } = await renderBar(<AuxTop />, withActiveProject());
      fireEvent.click(await screen.findByText("Controls", {}, TIMEOUT));
      await waitFor(() => expect(bottom(store).visible).toBe(true));
      expect(bottom(store).hover).toBe(true);
    });

    it("should drop the hover on a second toggle, keeping the drawer pinned", async () => {
      const { store } = await renderBar(<AuxTop />, withActiveProject());
      const button = await screen.findByText("Controls", {}, TIMEOUT);
      fireEvent.click(button);
      fireEvent.click(button);
      await waitFor(() => expect(bottom(store).hover).toBe(false));
      expect(bottom(store).visible).toBe(true);
    });
  });
});
