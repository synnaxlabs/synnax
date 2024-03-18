import { alamos } from "@synnaxlabs/alamos";
import {
  DataType,
  Frame,
  Series,
  TimeRange,
  TimeSpan,
  channel,
} from "@synnaxlabs/client";
import { type Mock, describe, expect, it, vi } from "vitest";

import { Cache } from "@/telem/client/cache/cache";
import { type ReadRemoteFunc, Reader } from "@/telem/client/reader";

class MockRetriever implements channel.Retriever {
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
    const { normalized } = channel.analyzeParams(channels);
    return normalized.map(
      (key) =>
        new channel.Channel({
          key: key as number,
          name: `channel-${key}`,
          dataType: DataType.FLOAT32,
          isIndex: false,
        }),
    );
  }
}

const basicRemoteReadFunc =
  (fn: Mock): ReadRemoteFunc =>
  async (tr, keys) => {
    fn(tr, keys);
    return new Frame(
      keys,
      keys.map(
        () =>
          new Series({
            data: new Float32Array([1, 2, 3]),
            alignment: 0,
            timeRange: tr,
          }),
      ),
    );
  };

const retriever = new channel.DebouncedBatchRetriever(new MockRetriever(), 10);

describe("channelRetriever", () => {
  it("should correctly execute a simple read", async () => {
    const manager = new Cache(retriever, alamos.NOOP);
    const remoteReadF = vi.fn();
    const reader = new Reader(manager, basicRemoteReadFunc(remoteReadF), alamos.NOOP);
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const res = await reader.read(tr, [1, 2]);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(remoteReadF).toHaveBeenCalledWith(tr, [1, 2]);
    expect(Object.keys(res)).toHaveLength(2);
    expect(res[1].data).toHaveLength(1);
    expect(res[2].data).toHaveLength(1);
    expect(res[1].data[0].at(0)).toBe(1);
    expect(res[2].data[0].at(0)).toBe(1);
    expect(() => manager.get(1)).not.toThrow();
    expect(() => manager.get(2)).not.toThrow();
  });
  it("should skip a read if the value is in the cache", async () => {
    const manager = new Cache(retriever, alamos.NOOP);
    const remoteReadF = vi.fn();
    const reader = new Reader(manager, basicRemoteReadFunc(remoteReadF), alamos.NOOP);
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const res = await reader.read(tr, [1, 2]);
    expect(remoteReadF).toHaveBeenCalledWith(tr, [1, 2]);
    expect(Object.keys(res)).toHaveLength(2);
    expect(res[1].data).toHaveLength(1);
    expect(res[2].data).toHaveLength(1);
    expect(res[1].data[0].at(0)).toBe(1);
    expect(res[2].data[0].at(0)).toBe(1);
    const res2 = await reader.read(tr, [1, 2]);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(Object.keys(res2)).toHaveLength(2);
    expect(res2[1].data).toHaveLength(1);
    expect(res2[2].data).toHaveLength(1);
    expect(res2[1].data[0].at(0)).toBe(1);
    expect(res2[2].data[0].at(0)).toBe(1);
  });
  it("should correctly batch multiple read requests with exactly the same time range", async () => {
    const manager = new Cache(retriever, alamos.NOOP);
    const remoteReadF = vi.fn();
    const reader = new Reader(manager, basicRemoteReadFunc(remoteReadF), alamos.NOOP);
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const res2 = await Promise.all([
      reader.read(tr, [1, 2]),
      reader.read(tr, [3, 4, 5]),
    ]);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(remoteReadF).toHaveBeenCalledWith(tr, [1, 2, 3, 4, 5]);
    expect(res2).toHaveLength(2);
    expect(Object.keys(res2[0])).toHaveLength(2);
    expect(Object.keys(res2[1])).toHaveLength(3);
    expect(res2[0][1].data).toHaveLength(1);
    expect(res2[0][2].data).toHaveLength(1);
    expect(res2[1][3].data).toHaveLength(1);
    expect(res2[1][4].data).toHaveLength(1);
    expect(res2[1][5].data).toHaveLength(1);
  });
  it("should correclty batch multiple read requests with different time ranges", async () => {
    const manager = new Cache(retriever, alamos.NOOP);
    const remoteReadF = vi.fn();
    const reader = new Reader(manager, basicRemoteReadFunc(remoteReadF), alamos.NOOP);
    const tr1 = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const tr2 = new TimeRange(TimeSpan.seconds(2), TimeSpan.seconds(4));
    const res2 = await Promise.all([
      reader.read(tr1, [1, 2]),
      reader.read(tr2, [3, 4, 5]),
    ]);
    expect(remoteReadF).toHaveBeenCalledTimes(2);
    expect(remoteReadF).toHaveBeenCalledWith(tr1, [1, 2]);
    expect(remoteReadF).toHaveBeenCalledWith(tr2, [3, 4, 5]);
    expect(res2).toHaveLength(2);
    expect(Object.keys(res2[0])).toHaveLength(2);
    expect(Object.keys(res2[1])).toHaveLength(3);
    expect(res2[0][1].data).toHaveLength(1);
    expect(res2[0][2].data).toHaveLength(1);
    expect(res2[1][3].data).toHaveLength(1);
    expect(res2[1][4].data).toHaveLength(1);
    expect(res2[1][5].data).toHaveLength(1);
  });
  it("should correctly batch multiple read requests with time ranges within 5 milliseconds of each other", async () => {
    const manager = new Cache(retriever, alamos.NOOP);
    const remoteReadF = vi.fn();
    const reader = new Reader(manager, basicRemoteReadFunc(remoteReadF), alamos.NOOP);
    const tr1 = new TimeRange(TimeSpan.milliseconds(999), TimeSpan.seconds(1));
    const tr2 = new TimeRange(TimeSpan.milliseconds(998), TimeSpan.seconds(1));
    const res2 = await Promise.all([
      reader.read(tr1, [1, 2]),
      reader.read(tr2, [3, 4, 5]),
    ]);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(remoteReadF).toHaveBeenCalledWith(tr1, [1, 2, 3, 4, 5]);
    expect(res2).toHaveLength(2);
    expect(Object.keys(res2[0])).toHaveLength(2);
    expect(Object.keys(res2[1])).toHaveLength(3);
    expect(res2[0][1].data).toHaveLength(1);
    expect(res2[0][2].data).toHaveLength(1);
    expect(res2[1][3].data).toHaveLength(1);
    expect(res2[1][4].data).toHaveLength(1);
    expect(res2[1][5].data).toHaveLength(1);
  });
});
