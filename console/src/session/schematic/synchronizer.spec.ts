// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project, type schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Session } from "@/session";
import { renderHookWithConsole, uniqueName } from "@/testutil";

const client = createTestClient();

let proj: project.Project;

beforeAll(async () => {
  proj = await client.projects.create({ name: uniqueName("project"), layout: {} });
  // Epoch events only fire on a live change stream; open it up front so the
  // mount-time sweep runs deterministically.
  await client.cache.ensureStreaming();
});

const createSchematic = async (): Promise<schematic.Schematic> =>
  await client.schematics.create(proj.key, { name: uniqueName("schematic") });

const preloadWith = (...keys: string[]): Partial<Session.State> => ({
  [Session.Schematic.SLICE_NAME]: {
    ...Session.Schematic.ZERO_SLICE_STATE,
    schematics: Object.fromEntries(
      keys.map((key) => [key, Session.Schematic.ZERO_STATE]),
    ),
  },
});

describe("Schematic.SYNCHRONIZERS", () => {
  it("prunes session state when its schematic is deleted while connected", async () => {
    const created = await createSchematic();
    const { store } = await renderHookWithConsole(
      () => Session.Synchronizer.use(Session.Schematic.SYNCHRONIZERS),
      { client, preloadedState: preloadWith(created.key) },
    );
    expect(store.getState().schematic.schematics[created.key]).toBeDefined();
    await client.schematics.delete(created.key);
    await waitFor(() => {
      expect(store.getState().schematic.schematics[created.key]).toBeUndefined();
    });
  });

  it("sweeps references to schematics that vanished while away, keeping live ones", async () => {
    const survivor = await createSchematic();
    const ghost = uuid.create();
    const { store } = await renderHookWithConsole(
      () => Session.Synchronizer.use(Session.Schematic.SYNCHRONIZERS),
      { client, preloadedState: preloadWith(survivor.key, ghost) },
    );
    await waitFor(() => {
      expect(store.getState().schematic.schematics[ghost]).toBeUndefined();
    });
    expect(store.getState().schematic.schematics[survivor.key]).toBeDefined();
  });
});
