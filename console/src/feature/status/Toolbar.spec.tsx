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
import { uuid } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Status } from "@/feature/status";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import {
  assertDefined,
  createConsoleWrapper,
  createTestClientWithGrants,
  queryIcon,
  renderHookWithConsole,
  resolveFocusedTab,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

beforeAll(async () => {
  await client.connect();
});

// Favorite eviction lives in the status synchronizer, mounted app-wide in
// production; the toolbar only renders what survives.
const Synchronizers = (): null => {
  Session.Synchronizer.use(Session.Status.SYNCHRONIZERS);
  return null;
};

const renderToolbar = async (
  favorites: status.Key[] = [],
  as: Client = client,
): Promise<{ store: TestStore; container: HTMLElement }> => {
  const { wrapper, store } = await createConsoleWrapper({
    client: as,
    preloadedState: {
      [Session.Status.SLICE_NAME]: { version: 0, favorites },
    },
  });
  const { container } = render(
    <>
      <Synchronizers />
      {Status.TOOLBAR.content}
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { store, container };
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
    const { store } = await renderToolbar();
    store.dispatch(Session.Project.select(proj.key));
    fireEvent.click(await screen.findByText("Open status explorer"));
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
    const { store } = await renderToolbar([uuid.create(), s.key]);
    expect(await screen.findByText(s.name)).toBeTruthy();
    await waitFor(() =>
      expect(Session.Status.selectFavorites(store.getState())).toEqual([s.key]),
    );
  });
});

describe("status toolbar permissions", () => {
  it("should hide the toolbar from a subject who cannot read statuses", async () => {
    const denied = await createTestClientWithGrants(client);
    assertDefined(Status.TOOLBAR.useVisible);
    const { result } = await renderHookWithConsole(Status.TOOLBAR.useVisible, {
      client: denied,
    });
    await waitFor(() => expect(result.current).toBe(false));
  });

  it("should offer the toolbar to a viewer, who may read statuses", async () => {
    assertDefined(Status.TOOLBAR.useVisible);
    const { result } = await renderHookWithConsole(Status.TOOLBAR.useVisible, {
      client: await roles.get("Viewer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should withhold the create action from a viewer", async () => {
    const { container } = await renderToolbar([], await roles.get("Viewer"));
    await screen.findByText("No favorited statuses");
    await waitFor(() => expect(queryIcon(container, "explore")).toBeTruthy());
    expect(queryIcon(container, "add")).toBeNull();
  });

  it("should offer the create action to an engineer", async () => {
    const { container } = await renderToolbar([], await roles.get("Engineer"));
    await screen.findByText("No favorited statuses");
    await waitFor(() => expect(queryIcon(container, "add")).toBeTruthy());
  });
});
