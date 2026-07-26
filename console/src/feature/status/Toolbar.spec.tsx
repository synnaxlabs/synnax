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
import { uuid } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Status } from "@/feature/status";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  resolveFocusedTab,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

beforeAll(async () => {
  // Epoch events only fire on a live change stream; open it up front so the
  // mount-time favorites sweep runs deterministically.
  await client.cache.ensureStreaming();
});

// Favorite eviction lives in the status synchronizer, mounted app-wide in
// production; the toolbar only renders what survives.
const Synchronizers = (): null => {
  Session.Synchronizer.use(Session.Status.SYNCHRONIZERS);
  return null;
};

const renderToolbar = async (favorites: status.Key[] = []): Promise<TestStore> => {
  const { wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: {
      [Session.Status.SLICE_NAME]: { version: 0, favorites },
    },
  });
  render(
    <>
      <Synchronizers />
      {Status.TOOLBAR.content}
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return store;
};

const createStatus = async (message = "") =>
  await client.statuses.set(
    status.create({ name: uniqueName("status"), variant: "error", message }),
  );

describe("status toolbar", () => {
  it("should open the explorer from the empty state action", async () => {
    const proj = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    const store = await renderToolbar();
    store.dispatch(Session.Project.select(proj.key));
    fireEvent.click(await screen.findByText("Open Status Explorer"));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.type).toBe(Status.Explorer.TAB_TYPE);
  });

  it("should render favorited statuses with their message", async () => {
    const s = await createStatus("thermocouple offline");
    await renderToolbar([s.key]);
    expect(await screen.findByText(s.name)).toBeTruthy();
    expect(screen.getByText("thermocouple offline")).toBeTruthy();
  });

  it("should drop favorites whose status no longer exists", async () => {
    const s = await createStatus();
    const store = await renderToolbar([uuid.create(), s.key]);
    expect(await screen.findByText(s.name)).toBeTruthy();
    await waitFor(() =>
      expect(Session.Status.selectFavorites(store.getState())).toEqual([s.key]),
    );
  });
});
