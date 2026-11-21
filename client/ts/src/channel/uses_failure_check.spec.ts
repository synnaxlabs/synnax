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
import { type z } from "zod";

import { batch } from "@/batch";
import { channel } from "@/channel";
import { framer } from "@/framer";
import { query } from "@/query";
import { status } from "@/status";
import { PushStreamer, TEST_CORE_ADDRESS } from "@/testutil";

const FAILED: channel.Payload = {
  key: 1,
  name: "failed",
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
const LATER: channel.Payload = { ...FAILED, key: 2, name: "later" };

/**
 * Creates a channel client on a fake transport. The next `core.failUses` retrieves that
 * ask for uses fail with an error that is not a connection error.
 */
const createFake = async () => {
  const usesRetrieves: channel.Key[][] = [];
  const core = { failUses: 0 };
  const unary: UnaryClient = {
    use: () => {},
    send: async <RQ extends z.ZodType, RS extends z.ZodType = RQ>(
      target: string,
      req: z.input<RQ> | z.infer<RQ>,
      _: RQ,
      resSchema: RS,
    ): Promise<z.infer<RS>> => {
      if (target !== "/channel/retrieve")
        throw new Error(`unexpected request to ${target}`);
      const { includeUses, where } = channel.retrieveRequestZ.parse(req);
      const keys = where?.key?.in;
      if (includeUses === true && keys != null) {
        usesRetrieves.push(keys);
        if (core.failUses > 0) {
          core.failUses--;
          throw new Error("internal server error");
        }
        const used = [FAILED, LATER].filter((p) => keys.includes(p.key));
        return resSchema.parse({ channels: used.map((p) => ({ ...p, uses: [] })) });
      }
      return resSchema.parse({ channels: [FAILED, LATER] });
    },
  };
  const stream = new PushStreamer();
  const cache = new query.Cache({
    openStreamer: async () => stream,
    breaker: { baseInterval: TimeSpan.milliseconds(1) },
    onError: () => {},
  });
  onTestFinished(async () => await cache.close());
  const coreURL = new url.URL(TEST_CORE_ADDRESS);
  const channels = new channel.Client({
    framer: new framer.Client({
      stream: new WebSocketClient({ endpoint: coreURL, codec: binary.JSON_CODEC }),
      unary,
      file: new HTTPClient({ endpoint: coreURL, codec: binary.JSON_CODEC }),
      retrieveChannels: async () => [],
    }),
    unary,
    writer: new channel.Writer(unary),
    statuses: new status.Client({ unary, cache }),
    cache,
    batch: new batch.Client(unary),
  });
  await cache.ensureStreaming();
  const change = (...keys: channel.Key[]): void =>
    stream.push(
      new framer.Frame({
        [query.REFERENCE_CHANGE_CHANNEL_NAME]: new Series(
          keys.map((key) => `${channel.TYPE_RESOURCE_ID.type}:${key}`),
        ),
      }),
    );
  return { channels, core, usesRetrieves, change };
};

describe("Channel uses refetch after a failure", () => {
  it("should fetch the uses of a failed refetch again at the next change", async () => {
    const { channels, core, usesRetrieves, change } = await createFake();
    const q = { where: { key: { in: [FAILED.key, LATER.key] } }, includeUses: true };
    const list = channels.watch(q);
    onTestFinished(list.onChange(() => {}));
    await list.retrieve();
    expect(usesRetrieves).toEqual([[FAILED.key, LATER.key]]);
    core.failUses = 1;
    change(FAILED.key);
    await expect.poll(() => usesRetrieves.length).toBe(2);
    expect(usesRetrieves[1]).toEqual([FAILED.key]);
    change(LATER.key);
    await expect.poll(() => usesRetrieves.length).toBe(3);
    expect(usesRetrieves[2]).toEqual(expect.arrayContaining([FAILED.key, LATER.key]));
  });
});
