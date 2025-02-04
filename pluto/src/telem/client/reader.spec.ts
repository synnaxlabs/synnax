// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import {
  channel,
  DataType,
  Frame,
  Series,
  TimeRange,
  TimeSpan,
} from "@synnaxlabs/client";
import { describe, expect, it, type Mock, vi } from "vitest";

import { Cache } from "@/telem/client/cache/cache";
import { Reader, type ReadRemoteFunc } from "@/telem/client/reader";

class MockRetriever implements channel.Retriever {
  async search(): Promise<channel.Payload[]> {
    throw new Error("Method not implemented.");
  }

  async page(): Promise<channel.Payload[]> {
    throw new Error("Method not implemented.");
  }

  async retrieve(channels: channel.Params): Promise<channel.Payload[]> {
    const { normalized } = channel.analyzeChannelParams(channels);
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
            alignment: 0n,
            timeRange: tr,
          }),
      ),
    );
  };

const retriever = new channel.DebouncedBatchRetriever(new MockRetriever(), 10);

const newCache = (): Cache =>
  new Cache({
    channelRetriever: retriever,
    instrumentation: alamos.NOOP,
  });

describe("channelRetriever", () => {
  it("should correctly execute a simple read", async () => {
    const cache = newCache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: basicRemoteReadFunc(remoteReadF),
      instrumentation: alamos.NOOP,
    });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const res = await reader.read(tr, [1, 2]);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(remoteReadF).toHaveBeenCalledWith(tr, [1, 2]);
    expect(Object.keys(res)).toHaveLength(2);
    expect(res[1].data).toHaveLength(1);
    expect(res[2].data).toHaveLength(1);
    expect(res[1].data[0].at(0)).toBe(1);
    expect(res[2].data[0].at(0)).toBe(1);
    expect(() => cache.get(1)).not.toThrow();
    expect(() => cache.get(2)).not.toThrow();
  });
  it("should skip a read if the value is in the cache", async () => {
    const cache = newCache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: basicRemoteReadFunc(remoteReadF),
      instrumentation: alamos.NOOP,
    });
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
    const cache = newCache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: basicRemoteReadFunc(remoteReadF),
    });
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
    const cache = newCache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: basicRemoteReadFunc(remoteReadF),
      instrumentation: alamos.NOOP,
    });
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
    const manager = newCache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache: manager,
      readRemote: basicRemoteReadFunc(remoteReadF),
    });
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
