// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, TimeStamp, uuid } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";

import { NotFoundError, ValidationError } from "@/index";
import { opcua } from "@/opcua";
import { query } from "@/query";
import { createTestClient, waitForStreamLive } from "@/testutil";

const client = createTestClient();
const other = createTestClient();

const ENDPOINT = "opc.tcp://localhost:4840";

describe("OPC UA server", async () => {
  await other.connect();
  const rack = await client.racks.create({ name: "opcua-server-test" });
  const create = async (name = "PLC"): Promise<opcua.server.Server> =>
    await client.opcua.servers.create({
      rack: rack.key,
      name,
      connection: { endpoint: ENDPOINT },
    });
  const rename = (s: opcua.server.Server, name: string): opcua.server.Update => ({
    group: "identity",
    rack: s.rack,
    name,
    connection: s.connection,
  });

  describe("create", () => {
    it("should store the server with the connection defaults", async () => {
      const s = await create();
      expect(s.connection.auth).toEqual({ type: "anonymous" });
      expect(s.connection.security).toEqual({ mode: "none" });
      const retrieved = await client.opcua.servers.retrieve({ key: s.key });
      expect(retrieved).toEqual(s);
    });

    it("should create several servers", async () => {
      const servers = await client.opcua.servers.create([
        { rack: rack.key, name: "A", connection: { endpoint: ENDPOINT } },
        { rack: rack.key, name: "B", connection: { endpoint: ENDPOINT } },
      ]);
      expect(servers.map((s) => s.name)).toEqual(["A", "B"]);
    });

    it("should throw a ValidationError for an endpoint without opc.tcp", async () => {
      await expect(
        client.opcua.servers.create({
          rack: rack.key,
          name: "Bad",
          connection: { endpoint: "http://localhost:4840" },
        }),
      ).rejects.toThrow(ValidationError);
    });

    it("should drop the optimistic server when the write fails", async () => {
      const key = uuid.create();
      const failure = new Error("write failed");
      await expect(
        client.opcua.servers.create(
          {
            key,
            rack: rack.key,
            name: "Optimistic",
            connection: { endpoint: ENDPOINT },
          },
          {
            onOptimistic: () => {
              throw failure;
            },
          },
        ),
      ).rejects.toBe(failure);
      expect(client.opcua.servers.watch(key).getCached()).toBeUndefined();
    });
  });

  describe("retrieve", () => {
    it("should include the status of the server", async () => {
      const s = await create();
      const retrieved = await client.opcua.servers.retrieve({
        key: s.key,
        includeStatus: true,
      });
      expect(retrieved.status?.owner).toEqual(opcua.server.resourceID(s.key));
      expect(retrieved.status?.details.server).toBe(s.key);
    });

    it("should update the status of a cached server", async () => {
      const s = await create();
      const params = { key: s.key, includeStatus: true };
      onTestFinished(client.opcua.servers.watch(params).onChange(() => {}));
      await client.opcua.servers.retrieve(params);
      await client.statuses.set({
        owner: opcua.server.resourceID(s.key),
        name: s.name,
        variant: "success",
        message: "Server connected",
        time: TimeStamp.now(),
        details: { rack: rack.key, server: s.key, connected: true },
      });
      await expect
        .poll(() => {
          const cached = client.opcua.servers.watch(params).getCached();
          return query.isLive(cached) ? cached.status?.message : undefined;
        })
        .toBe("Server connected");
    });

    it("should filter servers by rack", async () => {
      const s = await create();
      const servers = await client.opcua.servers.retrieve({ rack: rack.key });
      expect(servers.map(({ key }) => key)).toContain(s.key);
    });

    it("should throw a NotFoundError for a missing server", async () => {
      await expect(
        client.opcua.servers.retrieve({ key: uuid.create() }),
      ).rejects.toThrow(NotFoundError);
    });
  });

  describe("update", () => {
    it("should write the identity group and keep the bindings", async () => {
      const s = await create();
      const stored = await client.opcua.servers.update(s.key, rename(s, "Renamed"));
      const retrieved = await client.opcua.servers.retrieve({ key: s.key });
      expect(retrieved.name).toBe("Renamed");
      expect(stored).toEqual(retrieved);
    });

    it("should write the configuration group and keep the identity", async () => {
      const s = await create();
      const index = await client.channels.create({
        name: `opcua_time_${uuid.create()}`,
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data = await client.channels.create({
        name: `opcua_temp_${uuid.create()}`,
        dataType: DataType.FLOAT64,
        index: index.key,
      });
      const reads = [
        { node: "i=2258", field: "", index: index.key, channel: data.key },
      ];
      const stored = await client.opcua.servers.update(s.key, {
        group: "configuration",
        reads,
        writes: [],
      });
      const retrieved = await client.opcua.servers.retrieve({ key: s.key });
      expect(retrieved.reads).toEqual(reads);
      expect(retrieved.name).toBe(s.name);
      expect(stored).toEqual(retrieved);
    });

    it("should throw a NotFoundError for a missing server", async () => {
      const s = await create();
      await expect(
        client.opcua.servers.update(uuid.create(), rename(s, "Missing")),
      ).rejects.toThrow(NotFoundError);
    });
  });

  describe("delete", () => {
    it("should delete the server", async () => {
      const s = await create();
      await client.opcua.servers.delete(s.key);
      await expect(client.opcua.servers.retrieve({ key: s.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("cache", () => {
    it("should receive a server another client updates", async () => {
      const s = await create();
      await other.opcua.servers.retrieve({ key: s.key });
      await waitForStreamLive(other.connection);
      const seen: string[] = [];
      const stop = other.opcua.servers.onSet((server) => seen.push(server.name));
      await client.opcua.servers.update(s.key, rename(s, "Streamed"));
      await expect.poll(() => seen).toContain("Streamed");
      stop();
    });

    it("should drop a server another client deletes", async () => {
      const s = await create();
      await other.opcua.servers.retrieve({ key: s.key });
      await waitForStreamLive(other.connection);
      const deleted: string[] = [];
      const stop = other.opcua.servers.onDelete((key) => deleted.push(key));
      await client.opcua.servers.delete(s.key);
      await expect.poll(() => deleted).toContain(s.key);
      stop();
    });
  });
});
