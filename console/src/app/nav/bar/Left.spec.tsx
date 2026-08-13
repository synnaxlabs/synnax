// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Bar } from "@/app/nav/bar";
import { renderBar, withActiveProject } from "@/app/nav/bar/testutil";
import { Session } from "@/session";
import { getIconButton, type TestStore } from "@/testutil";

const navState = (window: Partial<Session.Nav.WindowState> = {}) => ({
  [Session.Nav.SLICE_NAME]: {
    windows: { [MAIN_WINDOW]: { ...Session.Nav.ZERO_WINDOW_STATE, ...window } },
  },
});

// Access-gated nav items appear only once their access query resolves against the
// cluster, so wait for the gated "range" item before asserting.
const findItem = async (
  container: HTMLElement,
  icon: string,
): Promise<HTMLButtonElement> => await waitFor(() => getIconButton(container, icon));

const left = (store: TestStore) => Session.Nav.selectWindowState(store.getState()).left;
const bottom = (store: TestStore) =>
  Session.Nav.selectWindowState(store.getState()).bottom;

describe("app/nav/bar/Left", () => {
  describe("rendering", () => {
    it("should render a nav button for each access-granted toolbar", async () => {
      const { container } = await renderBar(<Bar.Left />, withActiveProject());
      for (const icon of ["device", "task", "arc", "range", "channel", "project"])
        await findItem(container, icon);
    });

    it("should render the focused tab's toolbar in the bottom section", async () => {
      const { container } = await renderBar(<Bar.Left />, withActiveProject());
      await findItem(container, "component");
    });
  });

  describe("left selection", () => {
    it("should dispatch selectLeft and pin the item when clicked", async () => {
      const { container, store } = await renderBar(<Bar.Left />, withActiveProject());
      fireEvent.click(await findItem(container, "range"));
      await waitFor(() => expect(left(store).selected).toBe("range"));
      expect(left(store).hover).toBe(false);
    });

    it("should deselect when the pinned item is clicked again", async () => {
      const { container, store } = await renderBar(
        <Bar.Left />,
        withActiveProject(
          navState({
            left: { ...Session.Nav.ZERO_WINDOW_STATE.left, selected: "range" },
          }),
        ),
      );
      fireEvent.click(await findItem(container, "range"));
      await waitFor(() => expect(left(store).selected).toBeUndefined());
    });

    it("should mark the selected item with the selected class", async () => {
      const { container } = await renderBar(
        <Bar.Left />,
        withActiveProject(
          navState({
            left: { ...Session.Nav.ZERO_WINDOW_STATE.left, selected: "range" },
          }),
        ),
      );
      const button = await findItem(container, "range");
      expect(button.classList.contains("pluto--selected")).toBe(true);
    });
  });

  describe("bottom selection", () => {
    it("should dispatch selectBottom and show the bottom drawer when clicked", async () => {
      const { container, store } = await renderBar(<Bar.Left />, withActiveProject());
      fireEvent.click(await findItem(container, "component"));
      await waitFor(() => expect(bottom(store).visible).toBe(true));
    });

    it("should mark the bottom item selected when the drawer is visible", async () => {
      const { container } = await renderBar(
        <Bar.Left />,
        withActiveProject(
          navState({
            bottom: { ...Session.Nav.ZERO_WINDOW_STATE.bottom, visible: true },
          }),
        ),
      );
      const button = await findItem(container, "component");
      expect(button.classList.contains("pluto--selected")).toBe(true);
    });
  });
});
