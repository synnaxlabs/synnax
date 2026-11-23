// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { HTTPClient, type UnaryClient, WebSocketClient } from "@synnaxlabs/freighter";
import { binary, control, DataType, Series, TimeSpan, url } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";
import { z } from "zod";

import { batch } from "@/batch";
import { channel } from "@/channel";
import { ValidationError } from "@/errors";
import { framer } from "@/framer";
import { group } from "@/group";
import { query } from "@/query";
import { status } from "@/status";
import { PushStreamer, testCoreAddress } from "@/testutil";

const RENAMED: channel.Payload = {
  key: 1,
  name: "renamed",
  leaseholder: 1,
  dataType: DataType.FLOAT32,
  isIndex: false,
  index: 0,
  virtual: true,
  internal: false,
  expression: "",
  operations: [],
  concurrency: control.Concurrency.exclusive,
};
const OTHER: channel.Payload = { ...RENAMED, key: 2, name: "other" };
const SENTINEL: channel.Payload = { ...RENAMED, key: 3, name: "sentinel" };

const G1 = "5c2b3a4e-0f7d-4f5e-9a51-1d2a3b4c5d6e";
const G2 = "7e1f2a3b-4c5d-4e6f-8a9b-0c1d2e3f4a5b";
const CREATED_KEY = channel.keyZ.parse((3 << 20) | 7);
const CREATED: channel.Stored = {
  name: "created",
  leaseholder: 3,
  dataType: "float32",
  isIndex: false,
  localKey: 7,
  localIndex: 5,
  virtual: false,
  concurrency: control.Concurrency.exclusive,
  internal: false,
  operations: undefined,
  expression: "",
  group: undefined,
};

/** Returns a frame of another client's create of the channel. */
const createdFrame = (seq: number) =>
  new framer.Frame({
    [channel.ACTION_CHANNEL_NAME]: new Series([
      {
        key: CREATED_KEY,
        dispatchKey: `create-${seq}`,
        seq,
        actions: [{ type: "create", create: { channel: CREATED } }],
      },
    ]),
  });

const batchReqZ = z.object({
  requests: z.object({ payload: channel.dispatchReqZ }).array(),
});

interface StreamedClientParams {
  /** The actions the Core adds to each dispatch. */
  added?: channel.Action[];
  /** Settles before the Core answers a dispatch. */
  answered?: Promise<void>;
}

const createStreamedClient = async ({
  added = [],
  answered,
}: StreamedClientParams = {}) => {
  const dispatchKeys: string[] = [];
  const unary: UnaryClient = {
    use: () => {},
    send: async <RQ extends z.ZodType, RS extends z.ZodType = RQ>(
      target: string,
      req: z.input<RQ> | z.infer<RQ>,
      _: RQ,
      resSchema: RS,
    ): Promise<z.infer<RS>> => {
      if (target === "/channel/retrieve")
        return resSchema.parse({ channels: [RENAMED, OTHER, SENTINEL] });
      if (target === "/channel/dispatch") {
        dispatchKeys.push(channel.dispatchReqZ.parse(req).dispatchKey);
        await answered;
        return resSchema.parse({ actions: added });
      }
      if (target === "/batch") {
        const { requests } = batchReqZ.parse(req);
        dispatchKeys.push(...requests.map(({ payload }) => payload.dispatchKey));
        return resSchema.parse({ responses: requests.map(() => ({ actions: added })) });
      }
      throw new Error(`unexpected request to ${target}`);
    },
  };
  const stream = new PushStreamer();
  const batches = new batch.Client(unary);
  const cache = new query.Cache({
    openStreamer: async () => stream,
    breaker: { baseInterval: TimeSpan.milliseconds(1) },
    onError: () => {},
  });
  onTestFinished(async () => await cache.close());
  const coreURL = new url.URL(testCoreAddress());
  const frames = new framer.Client({
    stream: new WebSocketClient({ endpoint: coreURL, codec: binary.JSON_CODEC }),
    unary,
    file: new HTTPClient({ endpoint: coreURL, codec: binary.JSON_CODEC }),
    retrieveChannels: async () => [],
  });
  const channels = new channel.Client({
    framer: frames,
    unary,
    writer: new channel.Writer(unary),
    statuses: new status.Client({ unary, cache }),
    cache,
    batch: batches,
  });
  await cache.ensureStreaming();
  // Frames apply in order, so the delete applies after the late echo.
  const pushLateEcho = async (key: channel.Key, echo: channel.Action[]) => {
    stream.push(
      new framer.Frame({
        [channel.ACTION_CHANNEL_NAME]: new Series([
          { key, dispatchKey: dispatchKeys[0], seq: 1, actions: echo },
        ]),
      }),
    );
    stream.push(
      new framer.Frame({
        [channel.DELETE_CHANNEL_NAME]: new Series({
          data: new Uint32Array([SENTINEL.key]),
        }),
      }),
    );
    await expect.poll(() => channels.store.has(SENTINEL.key)).toBe(false);
  };
  await channels.retrieve([RENAMED.key, OTHER.key, SENTINEL.key]);
  return { stream, channels, batches, dispatchKeys, pushLateEcho };
};

describe("channel.Client", () => {
  describe("two renames of one channel", () => {
    it("should keep the second name when the echo of the first arrives late", async () => {
      const { channels, pushLateEcho } = await createStreamedClient();
      await channels.rename(RENAMED.key, "first");
      await channels.rename(RENAMED.key, "second");
      await pushLateEcho(RENAMED.key, [channel.rename({ name: "first" })]);
      expect(channels.store.get(RENAMED.key)?.name).toEqual("second");
    });
  });

  describe("two renames of several channels", () => {
    it("should keep the second names when the echo of the first arrives late", async () => {
      const { channels, pushLateEcho } = await createStreamedClient();
      const keys = [RENAMED.key, OTHER.key];
      await channels.rename(keys, ["first", "first other"]);
      await channels.rename(keys, ["second", "second other"]);
      await pushLateEcho(RENAMED.key, [channel.rename({ name: "first" })]);
      expect(channels.store.get(RENAMED.key)?.name).toEqual("second");
    });
  });

  describe("two batch moves of several channels", () => {
    it("should keep the second group when the echo of the first arrives late", async () => {
      const { channels, batches, pushLateEcho } = await createStreamedClient();
      const keys = [RENAMED.key, OTHER.key];
      await batches.send(channels.setGroupRequest(keys, G1));
      await batches.send(channels.setGroupRequest(keys, G2));
      await pushLateEcho(RENAMED.key, group.moveActions(G1));
      expect(channels.store.get(RENAMED.key)?.group).toEqual(G2);
    });
  });

  describe("rename", () => {
    it("should keep the error of a refused rename of one channel", async () => {
      const answered = Promise.withResolvers<void>();
      const { channels } = await createStreamedClient({ answered: answered.promise });
      const renamed = channels.rename(RENAMED.key, "first");
      expect(channels.store.get(RENAMED.key)?.name).toEqual("first");
      const refused = new ValidationError("name is taken");
      answered.reject(refused);
      await expect(renamed).rejects.toBe(refused);
      expect(channels.store.get(RENAMED.key)?.name).toEqual(RENAMED.name);
    });

    it("should send nothing when the optimistic callback fails", async () => {
      const { channels, dispatchKeys } = await createStreamedClient();
      const onOptimistic = () => {
        throw new Error("optimistic failed");
      };
      await expect(
        channels.rename(RENAMED.key, "first", { onOptimistic }),
      ).rejects.toThrow("optimistic failed");
      expect(dispatchKeys).toEqual([]);
      expect(channels.store.get(RENAMED.key)?.name).toEqual(RENAMED.name);
    });
  });

  describe("dispatch", () => {
    it("should cache the fields the Core sets before it resolves", async () => {
      const { channels } = await createStreamedClient({
        added: [channel.set({ path: ["dataType"], value: DataType.INT64.toString() })],
      });
      await channels.dispatch(
        RENAMED.key,
        channel.set({ path: ["expression"], value: "return 1" }),
      );
      expect(channels.store.get(RENAMED.key)?.dataType).toEqual(DataType.INT64);
    });
  });

  describe("a channel another client creates", () => {
    it("should cache the channel with the keys its stored form encodes", async () => {
      const { stream, channels } = await createStreamedClient();
      stream.push(createdFrame(1));
      await expect.poll(() => channels.store.get(CREATED_KEY)?.name).toEqual("created");
      expect(channels.store.get(CREATED_KEY)?.index).toEqual((3 << 20) | 5);
    });

    it("should keep the cached channel when its create echoes again during a refused dispatch", async () => {
      const answered = Promise.withResolvers<void>();
      const { stream, channels } = await createStreamedClient({
        answered: answered.promise,
      });
      stream.push(createdFrame(1));
      await expect.poll(() => channels.store.has(CREATED_KEY)).toBe(true);
      const before = channels.store.get(CREATED_KEY);
      const sent = channels.dispatch(
        CREATED_KEY,
        channel.set({ path: ["expression"], value: "return 1" }),
      );
      stream.push(createdFrame(2));
      stream.push(
        new framer.Frame({
          [channel.DELETE_CHANNEL_NAME]: new Series({
            data: new Uint32Array([SENTINEL.key]),
          }),
        }),
      );
      await expect.poll(() => channels.store.has(SENTINEL.key)).toBe(false);
      answered.reject(new Error("refused"));
      await expect(sent).rejects.toThrow("refused");
      expect(channels.store.get(CREATED_KEY) === before).toBe(true);
    });
  });
});
