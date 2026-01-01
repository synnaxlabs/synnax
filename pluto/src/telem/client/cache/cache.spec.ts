// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { channel, DataType, UnexpectedError } from "@synnaxlabs/client";
import { array, errors } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { Cache } from "@/telem/client/cache/cache";

class MockRetriever implements channel.Retriever {
  func: (channels: channel.Params, rangeKey?: string) => Promise<channel.Payload[]>;

  constructor(
    func: (channels: channel.Params, rangeKey?: string) => Promise<channel.Payload[]>,
  ) {
    this.func = func;
  }

  async retrieve(
    channels: channel.Params | channel.RetrieveRequest,
    opts?: channel.RetrieveOptions,
  ): Promise<channel.Payload[]> {
    if (typeof channels === "object" && !Array.isArray(channels))
      throw new errors.NotImplemented();
    return await this.func(channels, opts?.rangeKey);
  }
}

describe("cacheManager", () => {
  describe("populateMissing", () => {
    it("should populate missing entries in the cache", async () => {
      const called = vi.fn();
      const ret = new MockRetriever(async (batch) => {
        called(batch);
        return array.toArray(batch).map(
          (key) =>
            new channel.Channel({
              key: key as number,
              name: `channel-${key as channel.Key}`,
              dataType: DataType.FLOAT32,
              isIndex: false,
            }),
        );
      });
      const retriever = new channel.DebouncedBatchRetriever(ret, 10);
      const manager = new Cache({
        channelRetriever: retriever,
        instrumentation: alamos.NOOP,
      });
      expect(() => manager.get(1)).toThrow(UnexpectedError);
      await manager.populateMissing([1, 2]);
      expect(manager.get(1)).toBeDefined();
      expect(manager.get(2)).toBeDefined();
    });
    it("should not overwrite existing entries in the cache", async () => {
      const called = vi.fn();
      const ret = new MockRetriever(async (batch) => {
        called(batch);
        return array.toArray(batch).map(
          (key) =>
            new channel.Channel({
              key: key as number,
              name: `channel-${key as channel.Key}`,
              dataType: DataType.FLOAT32,
              isIndex: false,
            }),
        );
      });

      const retriever = new channel.DebouncedBatchRetriever(ret, 10);
      const manager = new Cache({
        channelRetriever: retriever,
        instrumentation: alamos.NOOP,
      });
      await manager.populateMissing([1, 2]);
      const existing = manager.get(1);
      await manager.populateMissing([1, 2]);
      expect(manager.get(1)).toBe(existing);
    });
  });
});
