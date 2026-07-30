// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Session } from "@/session";
import { renderHookWithConsole, uniqueName } from "@/testutil";

const client = createTestClient();

beforeAll(async () => {
  // Epoch events only fire on a live change stream; connect up front so the
  // mount-time sweep runs deterministically.
  await client.connect();
});

const preloadWith = (selected: string): Partial<Session.State> => ({
  [Session.Project.SLICE_NAME]: {
    ...Session.Project.ZERO_SLICE_STATE,
    selected,
  },
});

describe("Project.SYNCHRONIZERS", () => {
  it("clears a selected project that no longer exists on the cluster", async () => {
    const ghost = uuid.create();
    const { store } = await renderHookWithConsole(
      () => Session.Synchronizer.use(Session.Project.SYNCHRONIZERS),
      { client, preloadedState: preloadWith(ghost) },
    );
    await waitFor(() => {
      expect(Session.Project.selectOptionalSelected(store.getState())).toBeUndefined();
    });
  });

  it("clears the selection when the selected project is deleted while connected", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const { store } = await renderHookWithConsole(
      () => Session.Synchronizer.use(Session.Project.SYNCHRONIZERS),
      { client, preloadedState: preloadWith(proj.key) },
    );
    expect(Session.Project.selectOptionalSelected(store.getState())).toBe(proj.key);
    await client.projects.delete(proj.key);
    await waitFor(() => {
      expect(Session.Project.selectOptionalSelected(store.getState())).toBeUndefined();
    });
  });
});
