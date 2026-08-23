// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { xy } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock } from "vitest";

import { List } from "@/feature/status/list";
import { Modals } from "@/platform/modals";
import { findModalButton } from "@/platform/tree/menuTestutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  renderSuspended,
  stubClipboardWriteText,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

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
  as: Client = client,
): Promise<TestStore> => {
  const { wrapper, store } = await createConsoleWrapper({
    client: as,
    preloadedState: {
      [Session.Status.SLICE_NAME]: { version: 0, favorites },
    },
  });
  await renderSuspended(
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
    fireEvent.click(await screen.findByText("Copy diagnostics"));
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
        await client.statuses.retrieve(s.key);
        return true;
      } catch {
        return false;
      }
    };
    await waitFor(async () => expect(await statusExists()).toBe(false));
  });
});

describe("status list context menu permissions", () => {
  it("should withhold rename and delete from a viewer", async () => {
    const s = await createStatus();
    await renderMenu([s.key], [], await roles.get("Viewer"));
    expect(await screen.findByText("Copy diagnostics")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("should offer rename and delete to an engineer", async () => {
    const s = await createStatus();
    await renderMenu([s.key], [], await roles.get("Engineer"));
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(await screen.findByText("Delete")).toBeTruthy();
  });
});
