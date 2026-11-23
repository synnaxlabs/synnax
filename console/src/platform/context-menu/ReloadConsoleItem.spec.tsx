// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Menu } from "@synnaxlabs/lyra/menu";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ContextMenu } from "@/platform/context-menu";
import { Session } from "@/session";
import { MemoryKV, renderWithConsole, type TestStore } from "@/testutil";

const render = async (store?: TestStore): Promise<TestStore> => {
  const { store: rendered } = await renderWithConsole(
    <Menu.Menu>
      <ContextMenu.ReloadConsoleItem />
    </Menu.Menu>,
    { store },
  );
  return rendered;
};

const reloading = (store: TestStore): boolean =>
  Drift.selectWindow(store.getState())?.stage === "reloading";

const createSavingStore = async (kv: MemoryKV): Promise<TestStore> =>
  await Session.createStore({
    runtime: new Drift.NoopRuntime(),
    enablePrerender: false,
    openKV: () => kv,
  });

describe("ContextMenu.ReloadConsoleItem", () => {
  it("moves the window into its reloading stage", async () => {
    const store = await render();
    expect(reloading(store)).toBe(false);
    fireEvent.click(screen.getByText("Reload Console"));
    await waitFor(() => expect(reloading(store)).toBe(true));
  });

  it("saves a change that waits on the debounce before the window reloads", async () => {
    const kv = new MemoryKV();
    const store = await render(await createSavingStore(kv));
    // The copy holds what storage held when the reload began.
    const atReload = new MemoryKV();
    let copied = false;
    const unsubscribe = store.subscribe(() => {
      if (copied || !reloading(store)) return;
      copied = true;
      kv.store.forEach((value, key) => atReload.store.set(key, value));
    });
    store.dispatch(Session.Theme.set("dark"));
    fireEvent.click(screen.getByText("Reload Console"));
    await waitFor(() => expect(reloading(store)).toBe(true));
    unsubscribe();
    const reopened = await createSavingStore(atReload);
    expect(reopened.getState()[Session.Theme.SLICE_NAME].mode).toBe("dark");
  });
});
