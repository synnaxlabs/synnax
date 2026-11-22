// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type Synnax } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/channel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const remote = createTestClient();

const COUNT = 1010;

const createChannels = async (): Promise<string> => {
  const prefix = `brk2_list_${id.create()}_`;
  await remote.channels.create(
    Array.from({ length: COUNT }, (_, i) => ({
      name: `${prefix}${i}`,
      dataType: DataType.FLOAT32,
      virtual: true,
    })),
  );
  return prefix;
};

// Scrolls a channel list to its end, as a channel select does.
const scrollToEnd = async (client: Synnax, prefix: string) => {
  const wrapper = await createAsyncSynnaxWrapper({ client });
  const { result } = renderHook(
    () => Channel.useList({ query: { where: { name: { startsWith: prefix } } } }),
    { wrapper },
  );
  for (
    let i = 0;
    i < 40 && !(result.current.answered && result.current.complete);
    i++
  ) {
    act(() => result.current.more());
    await waitFor(() => expect(result.current.variant).toBe("success"));
  }
  return { length: result.current.data.length, complete: result.current.complete };
};

describe("Channel.useList round 2 breaker", () => {
  it("should scroll to the end when the built-in keys already arrived", async () => {
    const prefix = await createChannels();
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    await client.channels.builtIns.retrieve();
    expect(await scrollToEnd(client, prefix)).toEqual({
      length: COUNT,
      complete: true,
    });
  }, 60000);

  it("should scroll to the end when the list is the first read of the keys", async () => {
    const prefix = await createChannels();
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    expect(await scrollToEnd(client, prefix)).toEqual({
      length: COUNT,
      complete: true,
    });
  }, 60000);
});
