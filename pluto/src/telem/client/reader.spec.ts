// Copyright 2026 Synnax Labs, Inc.
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
import { errors } from "@synnaxlabs/x";
import { describe, expect, it, type Mock, vi } from "vitest";

import { Cache } from "@/telem/client/cache/cache";
import { Reader, type ReadRemoteFunc } from "@/telem/client/reader";

export class MockRetriever implements channel.Retriever {
  async retrieve(
    channels: channel.Params | channel.RetrieveRequest,
  ): Promise<channel.Payload[]> {
    if (typeof channels === "object" && !Array.isArray(channels))
      throw new errors.NotImplemented();
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

describe("read", () => {
  it("should correctly execute a simple read", async () => {
    const cache = newCache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: basicRemoteReadFunc(remoteReadF),
      instrumentation: alamos.NOOP,
    });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const res = await reader.read(tr, 1);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(remoteReadF).toHaveBeenCalledWith(tr, [1]);
    expect(res.length).toEqual(3);
    expect(res.at(0)).toEqual(1);
    expect(() => cache.get(1)).not.toThrow();
    expect(() => cache.get(2)).toThrow();
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
    const res = await reader.read(tr, 1);
    expect(remoteReadF).toHaveBeenCalledWith(tr, [1]);
    expect(res).toHaveLength(3);
    expect(res.at(0)).toEqual(1);
    const res2 = await reader.read(tr, 1);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(res2).toHaveLength(3);
    expect(res2.at(0)).toEqual(1);
  });

  it("should correctly batch multiple read requests with exactly the same time range", async () => {
    const cache = newCache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: basicRemoteReadFunc(remoteReadF),
    });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const res = await Promise.all([
      reader.read(tr, 1),
      reader.read(tr, 2),
      reader.read(tr, 3),
      reader.read(tr, 4),
      reader.read(tr, 5),
    ]);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(remoteReadF).toHaveBeenCalledWith(tr, [1, 2, 3, 4, 5]);
    expect(res[0]).toHaveLength(3);
    expect(res[1]).toHaveLength(3);
    expect(res[2]).toHaveLength(3);
    expect(res[3]).toHaveLength(3);
    expect(res[4]).toHaveLength(3);
  });

  it("should correctly batch multiple read requests with different time ranges", async () => {
    const cache = newCache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: basicRemoteReadFunc(remoteReadF),
      instrumentation: alamos.NOOP,
    });
    const tr1 = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const tr2 = new TimeRange(TimeSpan.seconds(2), TimeSpan.seconds(4));
    const res = await Promise.all([
      reader.read(tr1, 1),
      reader.read(tr1, 2),
      reader.read(tr2, 3),
      reader.read(tr2, 4),
      reader.read(tr2, 5),
    ]);
    expect(remoteReadF).toHaveBeenCalledTimes(2);
    expect(remoteReadF).toHaveBeenCalledWith(tr1, [1, 2]);
    expect(remoteReadF).toHaveBeenCalledWith(tr2, [3, 4, 5]);
    expect(res[0]).toHaveLength(3);
    expect(res[0].timeRange.equals(tr1)).toBe(true);
    expect(res[1]).toHaveLength(3);
    expect(res[1].timeRange.equals(tr1)).toBe(true);
    expect(res[2]).toHaveLength(3);
    expect(res[2].timeRange.equals(tr2)).toBe(true);
    expect(res[3]).toHaveLength(3);
    expect(res[3].timeRange.equals(tr2)).toBe(true);
    expect(res[4]).toHaveLength(3);
    expect(res[4].timeRange.equals(tr2)).toBe(true);
  });

  it("should correctly batch multiple read requests with time ranges within 5 milliseconds of each other", async () => {
    const manager = newCache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache: manager,
      readRemote: basicRemoteReadFunc(remoteReadF),
    });
    const tr1 = new TimeRange(TimeSpan.milliseconds(999), TimeSpan.milliseconds(1001));
    const tr2 = new TimeRange(TimeSpan.milliseconds(998), TimeSpan.seconds(1));
    const res = await Promise.all([
      reader.read(tr1, 1),
      reader.read(tr1, 2),
      reader.read(tr2, 3),
      reader.read(tr2, 4),
      reader.read(tr2, 5),
    ]);
    // We expected the read time range to have the maximum breadth possible of the two
    // time ranges.
    const expectedReadTr = new TimeRange(
      TimeSpan.milliseconds(998),
      TimeSpan.milliseconds(1001),
    );
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(remoteReadF).toHaveBeenCalledWith(expectedReadTr, [1, 2, 3, 4, 5]);
    expect(res[0]).toHaveLength(3);
    expect(res[0].timeRange.equals(expectedReadTr)).toBe(true);
    expect(res[0].at(0)).toEqual(1);
    expect(res[1]).toHaveLength(3);
    expect(res[1].timeRange.equals(expectedReadTr)).toBe(true);
    expect(res[1].at(0)).toEqual(1);
    expect(res[2]).toHaveLength(3);
    expect(res[2].timeRange.equals(expectedReadTr)).toBe(true);
    expect(res[2].at(0)).toEqual(1);
    expect(res[3]).toHaveLength(3);
    expect(res[3].timeRange.equals(expectedReadTr)).toBe(true);
    expect(res[3].at(0)).toEqual(1);
    expect(res[4]).toHaveLength(3);
    expect(res[4].timeRange.equals(expectedReadTr)).toBe(true);
    expect(res[4].at(0)).toEqual(1);
  });
});
