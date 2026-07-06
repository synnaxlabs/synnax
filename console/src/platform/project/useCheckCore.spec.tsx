// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type Synnax as Client } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Project } from "@/platform/project";
import { Session } from "@/session";
import { renderHookWithConsole, renderWithConsole } from "@/testutil";

const client: Client = createTestClient();

const CheckCoreProbe = (): null => {
  Project.useCheckCore();
  return null;
};

const probeWithClient = (c: Client): ReactElement => (
  <Synnax.TestProvider client={c}>
    <CheckCoreProbe />
  </Synnax.TestProvider>
);

describe("Project.useCheckCore", () => {
  it("keeps the active project selected while the client is unchanged", async () => {
    const selected = id.create();
    const { store, rerender } = await renderHookWithConsole(
      () => Project.useCheckCore(),
      {
        client,
        preloadedState: {
          [Session.Project.SLICE_NAME]: {
            ...Session.Project.ZERO_SLICE_STATE,
            selected,
          },
        },
      },
    );
    expect(Session.Project.selectOptionalSelected(store.getState())).toBe(selected);

    rerender();
    expect(Session.Project.selectOptionalSelected(store.getState())).toBe(selected);
  });

  it("clears the selected project when the client changes", async () => {
    const selected = id.create();
    const other: Client = createTestClient();
    expect(other.key).not.toBe(client.key);
    const { store, rerender } = await renderWithConsole(probeWithClient(client), {
      preloadedState: {
        [Session.Project.SLICE_NAME]: {
          ...Session.Project.ZERO_SLICE_STATE,
          selected,
        },
      },
    });
    expect(Session.Project.selectOptionalSelected(store.getState())).toBe(selected);

    rerender(probeWithClient(other));
    expect(Session.Project.selectOptionalSelected(store.getState())).toBeUndefined();
  });
});
