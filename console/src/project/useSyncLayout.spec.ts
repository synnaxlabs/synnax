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

import { logout } from "@/cluster/services/logout";
import { Layout } from "@/layout";
import { SLICE_NAME } from "@/project/slice";
import { type Project, ZERO_SLICE_STATE } from "@/project/types";
import { useSyncLayout } from "@/project/useSyncLayout";
import { type ConsolePreloadedState, createConsoleWrapper } from "@/testUtils";

const client: Synnax = createTestClient();

const preloadWithActive = (active: Project): ConsolePreloadedState => ({
  [SLICE_NAME]: { ...ZERO_SLICE_STATE, active },
});

describe("useSyncLayout", () => {
  it("should save layout changes to the active project", async () => {
    const proj = await client.projects.create({
      name: `sync-${id.create()}`,
      layout: {},
    });
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: preloadWithActive({ key: proj.key, name: proj.name }),
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
        Layout.place({
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
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: preloadWithActive({ key: proj.key, name: proj.name }),
    });
    const { result } = renderHook(
      () => {
        useSyncLayout();
        return Status.useNotifications();
      },
      { wrapper },
    );

    act(() => logout(store.dispatch));
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 500));
    });

    expect(
      result.current.statuses.filter(({ message }) =>
        message.includes("project layout"),
      ),
    ).toHaveLength(0);
  });
});
