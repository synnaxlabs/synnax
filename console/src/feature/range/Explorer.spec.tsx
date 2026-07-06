// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError } from "@synnaxlabs/client";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { Range as CommonRange } from "@/platform/range";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import { enableEditing, findToolbarIconButton } from "@/platform/view/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore, waitForPlacedLayout } from "@/testutil";

const client = createTestClient();

const renderExplorer = async (): Promise<{ store: TestStore }> => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  render(
    <>
      <Range.Explorer
        layoutKey={Range.EXPLORER_LAYOUT_TYPE}
        visible
        focused
        onClose={() => {}}
      />
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
    expect(await screen.findByText("Save Locally")).toBeTruthy();
  });

  it("places the overview layout when a range is clicked", async () => {
    const rng = await createTestRange(client);
    const { store } = await renderExplorer();
    fireEvent.click(await revealRange(rng.name));
    const key = await waitForPlacedLayout(store, CommonRange.OVERVIEW_LAYOUT_TYPE);
    expect(key).toBe(rng.key);
  });

  describe("context menu", () => {
    it("places the overview layout from View details", async () => {
      const rng = await createTestRange(client);
      const { store } = await renderExplorer();
      fireEvent.contextMenu(await revealRange(rng.name));
      fireEvent.click(await screen.findByText("View details"));
      const key = await waitForPlacedLayout(store, CommonRange.OVERVIEW_LAYOUT_TYPE);
      expect(key).toBe(rng.key);
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
      expect(await screen.findByText("Save Locally")).toBeTruthy();
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
