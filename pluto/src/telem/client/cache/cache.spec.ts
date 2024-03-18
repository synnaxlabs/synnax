import { alamos } from "@synnaxlabs/alamos";
import { DataType, UnexpectedError, channel } from "@synnaxlabs/client";
import { toArray } from "@synnaxlabs/x";
import { describe, it, vi, expect } from "vitest";

import { Cache } from "@/telem/client/cache/cache";

class MockRetriever implements channel.Retriever {
  func: (channels: channel.Params, rangeKey?: string) => Promise<channel.Payload[]>;

  constructor(
    func: (channels: channel.Params, rangeKey?: string) => Promise<channel.Payload[]>,
  ) {
    this.func = func;
  }

  async search(term: string, rangeKey?: string): Promise<channel.Payload[]> {
    throw new Error("Method not implemented.");
  }

  async page(
    offset: number,
    limit: number,
    rangeKey?: string,
  ): Promise<channel.Payload[]> {
    throw new Error("Method not implemented.");
  }

  async retrieve(
    channels: channel.Params,
    rangeKey?: string,
  ): Promise<channel.Payload[]> {
    return await this.func(channels, rangeKey);
  }
}

describe("cacheManager", () => {
  describe("populateMissing", () => {
    it("should populate missing entries in the cache", async () => {
      const called = vi.fn();
      const ret = new MockRetriever(async (batch) => {
        called(batch);
        return toArray(batch).map(
          (key) =>
            new channel.Channel({
              key: key as number,
              name: `channel-${key}`,
              dataType: DataType.FLOAT32,
              isIndex: false,
            }),
        );
      });
      const retriever = new channel.DebouncedBatchRetriever(ret, 10);
      const manager = new Cache(retriever, alamos.NOOP);
      expect(() => manager.get(1)).toThrow(UnexpectedError);
      await manager.populateMissing([1, 2]);
      expect(manager.get(1)).toBeDefined();
      expect(manager.get(2)).toBeDefined();
    });
    it("should not overwrite existing entries in the cache", async () => {
      const called = vi.fn();
      const ret = new MockRetriever(async (batch) => {
        called(batch);
        return toArray(batch).map(
          (key) =>
            new channel.Channel({
              key: key as number,
              name: `channel-${key}`,
              dataType: DataType.FLOAT32,
              isIndex: false,
            }),
        );
      });

      const retriever = new channel.DebouncedBatchRetriever(ret, 10);
      const manager = new Cache(retriever, alamos.NOOP);
      await manager.populateMissing([1, 2]);
      const existing = manager.get(1);
      await manager.populateMissing([1, 2]);
      expect(manager.get(1)).toBe(existing);
    });
  });
});
