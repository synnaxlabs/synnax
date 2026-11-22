// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, query } from "@synnaxlabs/client";
import {
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Channel } from "@/feature/channel";
import { Tree } from "@/platform/tree";
import { createHostItems, HOST_ID, renderLiveTree } from "@/platform/tree/testutil";

const client = createTestClient();

const BUILT_INS_TARGET = "/channel/retrieve-built-ins";

const hostChannels = (...channels: { key: channel.Key }[]) =>
  createHostItems((c) => [
    Tree.createSource(
      c.channels,
      query.keysRequest(channels.map((ch) => ch.key)),
      (ch) => Tree.toResource("channel")(ch.payload),
    ),
  ]);

describe("channel/tree built-in keys across a reconnect", () => {
  it("should keep the name field editable after a reconnect follows a failed fetch", async () => {
    const ch = await client.channels.create({
      name: uniqueName("ch"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const { proxy, client: proxied } = await createProxiedTestClient({
      retry: FAST_RETRY,
      onInternalError: () => {},
    });
    await proxied.connect();
    await waitForStreamLive(proxied.connection);
    const fault = { on: true };
    const { unary } = proxied.transport;
    const send = unary.send.bind(unary);
    vi.spyOn(unary, "send").mockImplementation(async (...args) => {
      if (fault.on && args[0] === BUILT_INS_TARGET) throw new Error("refused");
      return await send(...args);
    });
    await renderLiveTree({
      client: proxied,
      root: HOST_ID,
      items: { ...Channel.TREE_ITEMS, ...hostChannels(ch) },
    });
    // A failed fetch leaves the rename enabled.
    await waitFor(() =>
      expect(screen.getByText(ch.name).className).toContain("lyra-text--editable"),
    );
    // The Core answers again, and the connection drops and comes back.
    fault.on = false;
    const epoch = proxied.connection.status.details.epoch;
    await proxy.sever();
    await proxy.restore();
    await waitFor(
      () => expect(proxied.connection.status.details.epoch).toBeGreaterThan(epoch),
      { timeout: 15000 },
    );
    // Another user renames the channel, so its tree item renders again.
    const renamed = uniqueName("renamed");
    await client.channels.rename(ch.key, renamed);
    await screen.findByText(renamed, undefined, { timeout: 8000 });
    await waitFor(
      () =>
        expect(screen.getByText(renamed).className).toContain("lyra-text--editable"),
      { timeout: 8000 },
    );
  }, 60000);
});
