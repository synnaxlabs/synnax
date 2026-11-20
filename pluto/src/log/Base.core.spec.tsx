// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color as clientColor, log } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { color, id } from "@synnaxlabs/x";
import { render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Base } from "@/log/Base";
import { createAetherSetup, DEFAULT_STATE, initialStateOf } from "@/log/testutil";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

// Partial Aether mock: only intercepts type "log" for controlling worker-computed
// state (empty, scrolling, selectionStart, etc.). All other Aether consumers
// (Status, Flux, etc.) get the real implementation via the test providers.
// Type assertions below follow existing vi.mock patterns (vitest doesn't expose
// module types from importOriginal without import() annotations, which lint forbids).
const mockAetherUse = vi.hoisted(() => vi.fn());
vi.mock("@/aether", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  const realAether = actual.Aether as Record<string, unknown>;
  return {
    ...actual,
    Aether: {
      ...realAether,
      use: (...args: unknown[]) => {
        const opts = args[0] as { type?: string } | undefined;
        if (opts?.type === "log") return mockAetherUse(...args);
        return (realAether.use as (...a: unknown[]) => unknown)(...args);
      },
    },
  };
});

const setupAether = createAetherSetup(mockAetherUse);

const getAetherInitialState = (): Record<string, unknown> =>
  initialStateOf(mockAetherUse);

describe("log/Base", () => {
  describe("channel colors", () => {
    it("should send Aether a named channel color and its changes", async () => {
      const client = createTestClient();
      const SynnaxWrapper = await createAsyncSynnaxWrapper({ client });
      await waitForStreamLive(client.connection);
      const created = await client.colors.create({
        name: `log-${id.create()}`,
        light: "#123456",
      });
      const { setState } = setupAether();
      render(
        <Triggers.Provider>
          <Base
            channels={[
              log.channelEntryZ.parse({
                channel: 1,
                color: { type: "named", key: created.key },
              }),
            ]}
          />
        </Triggers.Provider>,
        { wrapper: SynnaxWrapper },
      );
      const sent = (): unknown => {
        const update = setState.mock.calls.at(-1)?.[0];
        const state =
          typeof update === "function"
            ? update(DEFAULT_STATE)
            : getAetherInitialState();
        return state.channelColors;
      };
      await waitFor(() => expect(sent()).toEqual({ 1: color.construct("#123456") }));
      await client.colors.dispatch(
        created.key,
        clientColor.set({ path: ["light"], value: color.construct("#654321") }),
      );
      await waitFor(() => expect(sent()).toEqual({ 1: color.construct("#654321") }));
    });
  });
});
