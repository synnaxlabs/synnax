// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Session } from "@/session";
import { documentIn, inWindow } from "@/session/window/testutil";
import { renderHookWithConsole, uniqueName } from "@/testutil";

const client = createTestClient();

beforeAll(async () => {
  await client.connect();
});

const createICD = async (): Promise<icd.ICD> =>
  await client.icds.create({ name: uniqueName("icd"), messages: [] });

const preloadWith = (...keys: string[]): Partial<Session.State> => ({
  [Session.ICD.SLICE_NAME]: {
    ...Session.ICD.ZERO_SLICE_STATE,
    windows: inWindow(
      Object.fromEntries(keys.map((key) => [key, Session.ICD.ZERO_STATE])),
    ),
  },
});

describe("ICD.SYNCHRONIZERS", () => {
  it("should remove the session state of an ICD deleted while connected", async () => {
    const created = await createICD();
    const { store } = await renderHookWithConsole(
      () => Session.Synchronizer.use(Session.ICD.SYNCHRONIZERS),
      { client, preloadedState: preloadWith(created.key) },
    );
    expect(documentIn(store.getState().icd, created.key)).toBeDefined();
    await client.icds.delete(created.key);
    await waitFor(() => {
      expect(documentIn(store.getState().icd, created.key)).toBeUndefined();
    });
  });

  it("should remove the session state of an ICD deleted while away, keeping live ones", async () => {
    const survivor = await createICD();
    const ghost = uuid.create();
    const { store } = await renderHookWithConsole(
      () => Session.Synchronizer.use(Session.ICD.SYNCHRONIZERS),
      { client, preloadedState: preloadWith(survivor.key, ghost) },
    );
    await waitFor(() => {
      expect(documentIn(store.getState().icd, ghost)).toBeUndefined();
    });
    expect(documentIn(store.getState().icd, survivor.key)).toBeDefined();
  });
});
