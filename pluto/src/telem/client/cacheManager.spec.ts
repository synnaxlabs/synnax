import { alamos } from "@synnaxlabs/alamos";
import { DataType, UnexpectedError, channel } from "@synnaxlabs/client";
import { describe, it, vi, expect } from "vitest";

import { CacheManager } from "@/telem/client/cacheManager";
import {
  ChannelRetriever,
  type RetrieveRemoteFunc,
} from "@/telem/client/channelRetriever";

describe("cacheManager", () => {
  describe("populateMissing", () => {
    it("should populate missing entries in the cache", async () => {
      const called = vi.fn();
      const remoteRetrieve: RetrieveRemoteFunc = async (batch) => {
        called(batch);
        return batch.map(
          (key) =>
            new channel.Channel({
              key,
              name: `channel-${key}`,
              dataType: DataType.FLOAT32,
              isIndex: false,
            }),
        );
      };
      const retriever = new ChannelRetriever(remoteRetrieve);
      const manager = new CacheManager(retriever, alamos.NOOP);
      expect(() => manager.get(1)).toThrow(UnexpectedError);
      await manager.populateMissing([1, 2]);
      expect(manager.get(1)).toBeDefined();
      expect(manager.get(2)).toBeDefined();
    });
    it("should not overwrite existing entries in the cache", async () => {
      const called = vi.fn();
      const remoteRetrieve: RetrieveRemoteFunc = async (batch) => {
        called(batch);
        return batch.map(
          (key) =>
            new channel.Channel({
              key,
              name: `channel-${key}`,
              dataType: DataType.FLOAT32,
              isIndex: false,
            }),
        );
      };
      const retriever = new ChannelRetriever(remoteRetrieve);
      const manager = new CacheManager(retriever, alamos.NOOP);
      await manager.populateMissing([1, 2]);
      const existing = manager.get(1);
      await manager.populateMissing([1, 2]);
      expect(manager.get(1)).toBe(existing);
    });
  });
});
