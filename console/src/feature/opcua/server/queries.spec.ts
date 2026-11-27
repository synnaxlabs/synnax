// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { opcua } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OPCUA } from "@/feature/opcua";
import { createAsyncSynnaxWrapper } from "@/testutil";

const client = createTestClient();

describe("OPC UA server queries", async () => {
  const rack = await client.racks.create({ name: uniqueName("opcua_rack") });
  const index = await client.channels.create({
    name: uniqueName("opcua_time"),
    dataType: "timestamp",
    isIndex: true,
  });
  const data = await client.channels.create({
    name: uniqueName("opcua_temp"),
    dataType: "float64",
    index: index.key,
  });
  const createServer = async (): Promise<opcua.server.Server> =>
    await client.opcua.servers.create({
      rack: rack.key,
      name: uniqueName("opcua_server"),
      connection: { endpoint: "opc.tcp://localhost:4840" },
      reads: [{ node: "i=2258", field: "", index: index.key, channel: data.key }],
    });

  describe("useRetrieve", () => {
    it("should retrieve the server with its status", async () => {
      const server = await createServer();
      const wrapper = await createAsyncSynnaxWrapper({ client });
      const { result } = renderHook(
        () => OPCUA.Server.useRetrieve({ key: server.key }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.data?.name).toBe(server.name));
      expect(result.current.data?.status).toBeDefined();
    });
  });

  describe("useRename", () => {
    it("should rename the server and keep its connection and bindings", async () => {
      const server = await createServer();
      const wrapper = await createAsyncSynnaxWrapper({ client });
      const { result } = renderHook(() => OPCUA.Server.useRename(), { wrapper });
      const name = uniqueName("renamed");
      await act(async () => {
        await result.current.updateAsync({ key: server.key, name });
      });
      const stored = await client.opcua.servers.retrieve({ key: server.key });
      expect(stored.name).toBe(name);
      expect(stored.connection).toEqual(server.connection);
      expect(stored.reads).toEqual(server.reads);
    });
  });

  describe("useDelete", () => {
    it("should delete the server", async () => {
      const server = await createServer();
      const wrapper = await createAsyncSynnaxWrapper({ client });
      const { result } = renderHook(() => OPCUA.Server.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(server.key);
      });
      await expect(
        client.opcua.servers.retrieve({ where: { key: { eq: server.key } } }),
      ).resolves.toEqual([]);
    });
  });
});
