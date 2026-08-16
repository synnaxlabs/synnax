// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import { enableEditing, findToolbarIconButton } from "@/platform/view/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  resolveFocusedTab,
  selectTestProject,
  type TestStore,
} from "@/testutil";

const client = createTestClient();

const expectFocusedRange = async (store: TestStore, key: string): Promise<void> => {
  const tab = await resolveFocusedTab(store, client);
  if (tab.variant !== "resource")
    throw new Error("focused tab is not a range resource");
  expect(tab.resource.type).toBe(ranger.TYPE_ONTOLOGY_ID.type);
  expect(tab.resource.key).toBe(key);
};

const renderExplorer = async (): Promise<{ store: TestStore }> => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  await selectTestProject(store, client);
  render(
    <>
      <Range.Explorer.Explorer />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  await screen.findByText("All Ranges");
  return { store };
};

// The shared cluster accumulates ranges across runs, so a fresh range may fall outside
// the list's first page. Narrow to it through the view search before interacting.
const revealRange = async (name: string): Promise<HTMLElement> => {
  await enableEditing();
  const search = await waitFor(() =>
    screen.getByPlaceholderText<HTMLInputElement>("Search ranges..."),
  );
  fireEvent.change(search, { target: { value: name } });
  return await screen.findByText(name);
};

const rowOf = (el: HTMLElement): HTMLElement => {
  const row = el.closest<HTMLElement>(".pluto-list__item");
  if (row == null) throw new Error("list item not found");
  return row;
};

const checkboxOf = (el: HTMLElement): HTMLInputElement => {
  const box = rowOf(el).querySelector<HTMLInputElement>("input[type='checkbox']");
  if (box == null) throw new Error("checkbox not found");
  return box;
};

const focusedTabs = (store: TestStore): string[] => {
  const state = store.getState();
  const panelKey = Session.Panel.selectSelected(state);
  if (panelKey == null) return [];
  return Session.Panel.selectSelectedTabs(state, panelKey);
};

describe("range/Explorer", () => {
  it("lists ranges stored on the cluster", async () => {
    const rng = await createTestRange(client);
    await renderExplorer();
    expect(await revealRange(rng.name)).toBeTruthy();
  });

  it("opens the create range modal from the toolbar button", async () => {
    await renderExplorer();
    await enableEditing();
    fireEvent.click(await findToolbarIconButton("add"));
    expect(await screen.findByText("Save locally")).toBeTruthy();
  });

  it("opens the overview tab when a range is clicked", async () => {
    const rng = await createTestRange(client);
    const { store } = await renderExplorer();
    fireEvent.click(await revealRange(rng.name));
    await expectFocusedRange(store, rng.key);
  });

  describe("selection", () => {
    it.each([
      ["shift", { shiftKey: true }],
      ["control", { ctrlKey: true }],
      ["command", { metaKey: true }],
    ])("selects rather than opens when %s is held", async (_, modifier) => {
      const rng = await createTestRange(client);
      const { store } = await renderExplorer();
      const name = await revealRange(rng.name);
      const before = focusedTabs(store);
      fireEvent.click(name, modifier);
      await waitFor(() => expect(checkboxOf(name).checked).toBe(true));
      expect(focusedTabs(store)).toEqual(before);
    });

    it("selects without opening when the checkbox is clicked", async () => {
      const rng = await createTestRange(client);
      const { store } = await renderExplorer();
      const name = await revealRange(rng.name);
      const before = focusedTabs(store);
      fireEvent.click(checkboxOf(name));
      await waitFor(() => expect(checkboxOf(name).checked).toBe(true));
      expect(focusedTabs(store)).toEqual(before);
    });

    it("leaves the range unselected when clicked without a modifier", async () => {
      const rng = await createTestRange(client);
      const { store } = await renderExplorer();
      const name = await revealRange(rng.name);
      fireEvent.click(name);
      await expectFocusedRange(store, rng.key);
      expect(checkboxOf(name).checked).toBe(false);
    });
  });

  describe("context menu", () => {
    it("opens the overview tab from View details", async () => {
      const rng = await createTestRange(client);
      const { store } = await renderExplorer();
      fireEvent.contextMenu(await revealRange(rng.name));
      fireEvent.click(await screen.findByText("View details"));
      await expectFocusedRange(store, rng.key);
    });

    it("favorites the range into the session slice", async () => {
      const rng = await createTestRange(client);
      const { store } = await renderExplorer();
      fireEvent.contextMenu(await revealRange(rng.name));
      fireEvent.click(await screen.findByText("Favorite"));
      await waitFor(() => {
        const state = Session.Range.selectState(store.getState(), rng.key);
        expect(state?.persisted).toBe(true);
      });
    });

    it("renames the range through the rename modal", async () => {
      const rng = await createTestRange(client);
      await renderExplorer();
      fireEvent.contextMenu(await revealRange(rng.name));
      fireEvent.click(await screen.findByText("Rename"));
      const input = await waitFor(() => {
        const el = screen.getByPlaceholderText<HTMLInputElement>("Name");
        expect(el.value).toBe(rng.name);
        return el;
      });
      const renamed = uniqueRangeName("renamed");
      fireEvent.change(input, { target: { value: renamed } });
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await client.ranges.retrieve(rng.key)).name).toBe(renamed),
      );
    });

    it("opens the create modal for a child range", async () => {
      const rng = await createTestRange(client);
      await renderExplorer();
      fireEvent.contextMenu(await revealRange(rng.name));
      fireEvent.click(await screen.findByText("Create child range"));
      expect(await screen.findByText("Save locally")).toBeTruthy();
    });

    it("deletes the range from the cluster after confirmation", async () => {
      const rng = await createTestRange(client);
      const { store } = await renderExplorer();
      store.dispatch(
        Session.Range.add({
          key: rng.key,
          name: rng.name,
          persisted: true,
          variant: "static",
          timeRange: rng.timeRange.numeric,
        }),
      );
      fireEvent.contextMenu(await revealRange(rng.name));
      fireEvent.click(await screen.findByText("Delete"));
      await screen.findByText(`Are you sure you want to delete ${rng.name}?`);
      fireEvent.click(findButton("Delete"));
      await waitFor(() =>
        expect(Session.Range.selectState(store.getState(), rng.key)).toBeUndefined(),
      );
      await waitFor(async () => {
        await expect(client.ranges.retrieve(rng.key)).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
    });
  });
});
