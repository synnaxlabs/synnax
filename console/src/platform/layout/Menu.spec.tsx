// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Menu as PMenu, Text as PText } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/platform/layout";
import { Session } from "@/session";
import { createTestStore, renderWithConsole, type TestStore } from "@/testutil";

const place = (store: TestStore, key: string): void =>
  act(() => {
    store.dispatch(
      Session.Layout.place({
        key,
        type: "menu-test",
        name: key,
        location: "mosaic",
        windowKey: Drift.MAIN_WINDOW,
        window: { title: key },
      }),
    );
  });

const renderMenu = async (layoutKey: string, extraKeys: string[] = []) => {
  const store = await createTestStore();
  place(store, layoutKey);
  extraKeys.forEach((k) => place(store, k));
  const result = await renderWithConsole(
    <PMenu.Menu>
      <Layout.MenuItems layoutKey={layoutKey} />
    </PMenu.Menu>,
    { store },
  );
  return { ...result, store };
};

const mosaicRoot = (store: TestStore) =>
  store.getState()[Session.Layout.SLICE_NAME].mosaics[Drift.MAIN_WINDOW].root;

describe("layout MenuItems", () => {
  it("renders the focus and close items for a mosaic layout", async () => {
    await renderMenu("l1");
    expect(screen.getByText("Focus")).toBeTruthy();
    expect(screen.getByText("Close")).toBeTruthy();
    expect(screen.getByText("Rename")).toBeTruthy();
  });

  it("does not render the open-in-new-window item outside of Tauri", async () => {
    await renderMenu("l1");
    expect(screen.queryByText("Open in new window")).toBeNull();
  });

  it("does not render the move-to-main-window item from the main window", async () => {
    await renderMenu("l1");
    expect(screen.queryByText("Move to main window")).toBeNull();
  });

  it("removes the layout from the store when the close item is clicked", async () => {
    const { store } = await renderMenu("l1");
    fireEvent.click(screen.getByText("Close"));
    await waitFor(() =>
      expect(Session.Layout.select(store.getState(), "l1")).toBeUndefined(),
    );
  });

  it("focuses the layout when the focus item is clicked", async () => {
    const { store } = await renderMenu("l1");
    fireEvent.click(screen.getByText("Focus"));
    await waitFor(() =>
      expect(
        store.getState()[Session.Layout.SLICE_NAME].mosaics[Drift.MAIN_WINDOW].focused,
      ).toBe("l1"),
    );
  });

  it("does not render split items for a single-tab mosaic", async () => {
    await renderMenu("l1");
    expect(screen.queryByText("Split horizontally")).toBeNull();
  });

  it("splits the mosaic when the split item is clicked on a multi-tab mosaic", async () => {
    const { store } = await renderMenu("l1", ["l2"]);
    expect(mosaicRoot(store).first).toBeUndefined();
    fireEvent.click(screen.getByText("Split horizontally"));
    await waitFor(() => {
      const root = mosaicRoot(store);
      expect(root.first).toBeDefined();
      expect(root.last).toBeDefined();
      expect(root.direction).toBe("x");
    });
  });

  it("splits the mosaic vertically when the vertical split item is clicked", async () => {
    const { store } = await renderMenu("l1", ["l2"]);
    fireEvent.click(screen.getByText("Split vertically"));
    await waitFor(() => {
      const root = mosaicRoot(store);
      expect(root.first).toBeDefined();
      expect(root.last).toBeDefined();
      expect(root.direction).toBe("y");
    });
  });

  it("puts the tab element into edit mode when the rename item is clicked", async () => {
    const store = await createTestStore();
    place(store, "l1");
    await renderWithConsole(
      <>
        <PText.Editable id="pluto-tab-l1" value="Tab One" onChange={vi.fn()} />
        <PMenu.Menu>
          <Layout.MenuItems layoutKey="l1" />
        </PMenu.Menu>
      </>,
      { store },
    );
    const editable = screen.getByText("Tab One");
    expect(editable.getAttribute("contenteditable")).toBe("false");
    fireEvent.click(screen.getByText("Rename"));
    await waitFor(() => expect(editable.getAttribute("contenteditable")).toBe("true"));
  });
});
