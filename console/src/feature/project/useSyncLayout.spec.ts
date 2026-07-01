// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type Synnax } from "@synnaxlabs/client";
import { Access, Status } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { useSyncLayout } from "@/feature/project/useSyncLayout";
import { Session } from "@/session";
import { type ConsolePreloadedState, createConsoleWrapper } from "@/testutil/testutil";

const client: Synnax = createTestClient();

const preloadWithActive = (selected: string): ConsolePreloadedState => ({
  [Session.Project.SLICE_NAME]: { ...Session.Project.ZERO_SLICE_STATE, selected },
});

describe("useSyncLayout", () => {
  it("should save layout changes to the active project", async () => {
    const proj = await client.projects.create({
      name: `sync-${id.create()}`,
      layout: {},
    });
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: preloadWithActive(proj.key),
    });
    const { result } = renderHook(
      () => {
        useSyncLayout();
        return Access.useLoadPermissions({});
      },
      { wrapper },
    );
    await waitFor(() => expect(result.current.variant).toEqual("success"), {
      timeout: 5000,
    });

    const layoutKey = id.create();
    act(() => {
      store.dispatch(
        Session.Layout.place({
          windowKey: "main",
          key: layoutKey,
          type: "schematic",
          name: "Operator",
          location: "mosaic",
        }),
      );
    });

    await waitFor(
      async () => {
        const retrieved = await client.projects.retrieve(proj.key);
        expect(JSON.stringify(retrieved.layout)).toContain(layoutKey);
      },
      { timeout: 5000 },
    );
  });

  it("should not surface an error when logout clears the active project", async () => {
    const proj = await client.projects.create({
      name: `sync-${id.create()}`,
      layout: {},
    });
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: preloadWithActive(proj.key),
    });
    const { result } = renderHook(
      () => {
        useSyncLayout();
        return {
          logout: Session.useLogout(),
          notifications: Status.useNotifications(),
        };
      },
      { wrapper },
    );

    act(() => result.current.logout());
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 500));
    });

    expect(
      result.current.notifications.statuses.filter(({ message }) =>
        message.includes("project layout"),
      ),
    ).toHaveLength(0);
  });
});
