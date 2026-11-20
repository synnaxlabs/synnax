// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  access,
  channel,
  DataType,
  framer,
  type Synnax as Client,
  user,
} from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { id, sleep, TimeSpan } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/channel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const createChannel = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: `deleted_${id.create()}`,
    dataType: DataType.FLOAT32,
    virtual: true,
  });

const renderDeleted = async (as: Client, key: channel.Key) => {
  const wrapper = await createAsyncSynnaxWrapper({ client: as });
  return renderHook(() => Channel.useDeleted(key), { wrapper });
};

describe("Channel.useDeleted", () => {
  it("should report a channel the Core deleted", async () => {
    const ch = await createChannel();
    await client.channels.delete(ch.key);
    const { result } = await renderDeleted(client, ch.key);
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should not report a channel that exists", async () => {
    const ch = await createChannel();
    const { result } = await renderDeleted(client, ch.key);
    await sleep.sleep(TimeSpan.milliseconds(300));
    expect(result.current).toBe(false);
  });

  it("should not report the zero key", async () => {
    const { result } = await renderDeleted(client, 0);
    expect(result.current).toBe(false);
  });

  it("should not report a channel the user cannot read", async () => {
    const [readable, hidden] = await Promise.all([createChannel(), createChannel()]);
    const viewer = await createTestClientWithPolicy(client, {
      name: id.create(),
      objects: [
        channel.resourceID(readable.key),
        framer.TYPE_RESOURCE_ID,
        user.TYPE_RESOURCE_ID,
        access.role.TYPE_RESOURCE_ID,
        access.policy.TYPE_RESOURCE_ID,
      ],
      actions: ["retrieve"],
    });
    const { result } = await renderDeleted(viewer, hidden.key);
    await sleep.sleep(TimeSpan.milliseconds(300));
    expect(result.current).toBe(false);
  });
});
