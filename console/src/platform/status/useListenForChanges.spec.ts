// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type status } from "@synnaxlabs/client";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Status } from "@/platform/status";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const createStatus = async (): Promise<status.Status> =>
  await client.statuses.set({
    name: uniqueName("status"),
    message: "listen test",
    variant: "success",
  });

const favoritesState = (keys: status.Key[]) => ({
  [Session.Status.SLICE_NAME]: { version: 0 as const, favorites: keys },
});

describe("Status.useListenForChanges", () => {
  it("should remove a favorite when its status is deleted remotely", async () => {
    const stat = await createStatus();
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: favoritesState([stat.key]),
    });
    renderHook(() => Status.useListenForChanges(), { wrapper });
    expect(Session.Status.selectFavorites(store.getState())).toContain(stat.key);
    await client.statuses.delete(stat.key);
    await waitFor(() => {
      expect(Session.Status.selectFavorites(store.getState())).not.toContain(stat.key);
    });
  });
});
