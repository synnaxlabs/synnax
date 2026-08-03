// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { xy } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock } from "vitest";

import { List } from "@/feature/status/list";
import { Modals } from "@/platform/modals";
import { findModalButton } from "@/platform/tree/menuTestutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  stubClipboardWriteText,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

let clipboard: Mock;

beforeEach(() => {
  clipboard = stubClipboardWriteText();
});

const createStatus = async (message = "under test") =>
  await client.statuses.set(
    status.create({ name: uniqueName("status"), variant: "error", message }),
  );

const renderMenu = async (
  keys: status.Key[],
  favorites: status.Key[] = [],
): Promise<TestStore> => {
  const { wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: {
      [Session.Status.SLICE_NAME]: { version: 0, favorites },
    },
  });
  render(
    <>
      {List.contextMenu({ keys, visible: true, position: xy.ZERO, cursor: xy.ZERO })}
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return store;
};

describe("status list context menu", () => {
  it("should favorite the selected statuses", async () => {
    const s = await createStatus();
    const store = await renderMenu([s.key]);
    fireEvent.click(await screen.findByText("Favorite"));
    await waitFor(() =>
      expect(Session.Status.selectFavorites(store.getState())).toEqual([s.key]),
    );
    expect(await screen.findByText("Unfavorite")).toBeTruthy();
  });

  it("should unfavorite an already-favorited status", async () => {
    const s = await createStatus();
    const store = await renderMenu([s.key], [s.key]);
    fireEvent.click(await screen.findByText("Unfavorite"));
    await waitFor(() =>
      expect(Session.Status.selectFavorites(store.getState())).toEqual([]),
    );
  });

  it("should copy the status diagnostics to the clipboard", async () => {
    const s = await createStatus("valve stuck open");
    await renderMenu([s.key]);
    fireEvent.click(await screen.findByText("Copy Diagnostics"));
    await waitFor(() => expect(clipboard).toHaveBeenCalledTimes(1));
    const copied = clipboard.mock.calls[0][0];
    expect(copied).toContain(s.name);
    expect(copied).toContain("valve stuck open");
  });

  it("should delete the status on the cluster after confirmation", async () => {
    const s = await createStatus();
    await renderMenu([s.key]);
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${s.name}?`);
    fireEvent.click(findModalButton("Delete"));
    const statusExists = async (): Promise<boolean> => {
      try {
        await client.statuses.retrieve({ key: s.key });
        return true;
      } catch {
        return false;
      }
    };
    await waitFor(async () => expect(await statusExists()).toBe(false));
  });

  it("should rename the status through the rename modal", async () => {
    const s = await createStatus();
    await renderMenu([s.key]);
    fireEvent.click(await screen.findByText("Rename"));
    const input = within(await screen.findByRole("dialog")).getByRole("textbox");
    const newName = uniqueName("renamed");
    fireEvent.change(input, { target: { value: newName } });
    fireEvent.click(findModalButton("Save"));
    await waitFor(async () => {
      const updated = await client.statuses.retrieve({ key: s.key });
      expect(updated.name).toBe(newName);
    });
  });
});
