// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Session } from "@/session";
import { renderHookWithConsole, uniqueName } from "@/testutil";

const client = createTestClient();

beforeAll(async () => {
  await client.connect();
});

const createStatus = async (): Promise<status.Status> =>
  await client.statuses.set({
    name: uniqueName("status"),
    message: "synchronizer test",
    variant: "success",
  });

const preloadWith = (...keys: status.Key[]): Partial<Session.State> => ({
  [Session.Status.SLICE_NAME]: { ...Session.Status.ZERO_SLICE_STATE, favorites: keys },
});

describe("Status.SYNCHRONIZERS", () => {
  it("should remove a favorite when its status is deleted while connected", async () => {
    const created = await createStatus();
    const { store } = await renderHookWithConsole(
      () => Session.Synchronizer.use(Session.Status.SYNCHRONIZERS),
      { client, preloadedState: preloadWith(created.key) },
    );
    expect(Session.Status.selectFavorites(store.getState())).toContain(created.key);
    await client.statuses.delete(created.key);
    await waitFor(() => {
      expect(Session.Status.selectFavorites(store.getState())).not.toContain(
        created.key,
      );
    });
  });

  it("should sweep favorites that vanished while away, keeping live ones", async () => {
    const survivor = await createStatus();
    const ghost = uuid.create();
    const { store } = await renderHookWithConsole(
      () => Session.Synchronizer.use(Session.Status.SYNCHRONIZERS),
      { client, preloadedState: preloadWith(survivor.key, ghost) },
    );
    await waitFor(() => {
      expect(Session.Status.selectFavorites(store.getState())).not.toContain(ghost);
    });
    expect(Session.Status.selectFavorites(store.getState())).toContain(survivor.key);
  });

  it("should keep a favorite created moments before the sweep", async () => {
    // Regression: reconcile once compared favorites against a listing of every status,
    // which could answer as of before this write and drop the favorite.
    const fresh = await createStatus();
    const { result, store } = await renderHookWithConsole(
      () => Session.Synchronizer.use(Session.Status.SYNCHRONIZERS),
      { client, preloadedState: preloadWith(fresh.key) },
    );
    await waitFor(() => expect(result.current).toBe(true));
    expect(Session.Status.selectFavorites(store.getState())).toContain(fresh.key);
  });
});
