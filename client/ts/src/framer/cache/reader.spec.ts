// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import { MultiSeries, TimeRange, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it, type Mock, vi } from "vitest";

import { UnexpectedError } from "@/errors";
import { Cache } from "@/framer/cache/cache";
import { Reader, type RemoteReader } from "@/framer/cache/reader";
import { Frame } from "@/framer/frame";
import { Series } from "@/index";

const basicRemoteReadFunc =
  (fn: Mock): RemoteReader =>
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

describe("read", () => {
  it("should correctly execute a simple read", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({ cache, readRemote: basicRemoteReadFunc(remoteReadF) });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const res = await reader.read(tr, 1);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(remoteReadF).toHaveBeenCalledWith(tr, [1]);
    expect(res.length).toEqual(3);
    expect(res.at(0)).toEqual(1);
    cache.close();
  });

  it("should not refetch a range that returned no data", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const readRemote: RemoteReader = async (tr, keys) => {
      remoteReadF(tr, keys);
      return new Frame([], []);
    };
    const reader = new Reader({ cache, readRemote });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const res = await reader.read(tr, 1);
    expect(res).toHaveLength(0);
    const res2 = await reader.read(tr, 1);
    expect(res2).toHaveLength(0);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    cache.close();
  });

  it("should refetch a range whose fetch failed", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const readRemote: RemoteReader = async (tr, keys) => {
      remoteReadF(tr, keys);
      throw new Error("connection lost");
    };
    const reader = new Reader({ cache, readRemote });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    await expect(reader.read(tr, 1)).rejects.toThrow("connection lost");
    await expect(reader.read(tr, 1)).rejects.toThrow("connection lost");
    expect(remoteReadF).toHaveBeenCalledTimes(2);
    cache.close();
  });

  it("should skip a read if the value is in the cache", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({ cache, readRemote: basicRemoteReadFunc(remoteReadF) });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const res = await reader.read(tr, 1);
    expect(remoteReadF).toHaveBeenCalledWith(tr, [1]);
    expect(res).toHaveLength(3);
    expect(res.at(0)).toEqual(1);
    const res2 = await reader.read(tr, 1);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(res2).toHaveLength(3);
    expect(res2.at(0)).toEqual(1);
    cache.close();
  });

  it("should correctly batch multiple read requests with exactly the same time range", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({ cache, readRemote: basicRemoteReadFunc(remoteReadF) });
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
    res.forEach((r) => expect(r).toHaveLength(3));
    cache.close();
  });

  it("should correctly batch multiple read requests with different time ranges", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({ cache, readRemote: basicRemoteReadFunc(remoteReadF) });
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
    expect(res[0].timeRange.equals(tr1)).toBe(true);
    expect(res[2].timeRange.equals(tr2)).toBe(true);
    cache.close();
  });

  it("should correctly batch multiple read requests with time ranges within 5 milliseconds of each other", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({ cache, readRemote: basicRemoteReadFunc(remoteReadF) });
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
    res.forEach((r) => {
      expect(r).toHaveLength(3);
      expect(r.timeRange.equals(expectedReadTr)).toBe(true);
      expect(r.at(0)).toEqual(1);
    });
    cache.close();
  });

  it("should fire a batch even while reads keep arriving", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: basicRemoteReadFunc(remoteReadF),
      batchDebounce: TimeSpan.milliseconds(30),
    });
    const readAt = (offset: number): Promise<unknown> =>
      reader.read(
        new TimeRange(TimeSpan.seconds(offset), TimeSpan.seconds(offset + 1)),
        offset + 1,
      );
    const pending = [readAt(0)];
    for (let i = 1; i <= 5; i++) {
      await new Promise((r) => setTimeout(r, 10));
      pending.push(readAt(i * 10));
    }
    // A timer reset by each read would still be waiting for 30ms of silence.
    expect(remoteReadF).toHaveBeenCalled();
    await Promise.all(pending);
    cache.close();
  });

  it("should skip fetching a gap that was filled while the read was queued", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: basicRemoteReadFunc(remoteReadF),
      batchDebounce: TimeSpan.milliseconds(20),
    });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const pending = reader.read(tr, 1);
    cache.get(1).writeStatic(
      new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3]),
          alignment: 0n,
          timeRange: tr,
        }),
      ]),
    );
    const res = await pending;
    expect(remoteReadF).not.toHaveBeenCalled();
    expect(res).toHaveLength(3);
    cache.close();
  });

  it("should narrow the fetch to the portion of the gap still missing", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: basicRemoteReadFunc(remoteReadF),
      batchDebounce: TimeSpan.milliseconds(20),
    });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(5));
    const cached = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const pending = reader.read(tr, 1);
    cache.get(1).writeStatic(
      new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3]),
          alignment: 0n,
          timeRange: cached,
        }),
      ]),
    );
    await pending;
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(remoteReadF).toHaveBeenCalledWith(
      new TimeRange(TimeSpan.seconds(3), TimeSpan.seconds(5)),
      [1],
    );
    cache.close();
  });

  // The leading buffer holds provisional leading alignments, so its span still
  // fetches: the fetched form is what pairs with other channels' fetched data.
  it("should fetch the range the live leading buffer covers", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({ cache, readRemote: basicRemoteReadFunc(remoteReadF) });
    cache.get(1).writeDynamic(
      new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3]),
          alignment: 0n,
          timeRange: new TimeRange(TimeSpan.seconds(10), TimeSpan.seconds(13)),
        }),
      ]),
    );
    const res = await reader.read(
      new TimeRange(TimeSpan.seconds(10), TimeSpan.seconds(20)),
      1,
    );
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(remoteReadF).toHaveBeenCalledWith(
      new TimeRange(TimeSpan.seconds(10), TimeSpan.seconds(20)),
      [1],
    );
    // Both representations return: 3 buffer samples plus the 3 fetched ones.
    expect(res.length).toEqual(6);
    cache.close();
  });

  it("should fetch the full range across the leading buffer start", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({ cache, readRemote: basicRemoteReadFunc(remoteReadF) });
    cache.get(1).writeDynamic(
      new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3]),
          alignment: 0n,
          timeRange: new TimeRange(TimeSpan.seconds(10), TimeSpan.seconds(13)),
        }),
      ]),
    );
    await reader.read(new TimeRange(TimeSpan.seconds(5), TimeSpan.seconds(20)), 1);
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    expect(remoteReadF).toHaveBeenCalledWith(
      new TimeRange(TimeSpan.seconds(5), TimeSpan.seconds(20)),
      [1],
    );
    cache.close();
  });

  it("should fail only the reads whose batch failed", async () => {
    const cache = new Cache();
    const failing = new TimeRange(TimeSpan.seconds(10), TimeSpan.seconds(12));
    const readRemote: RemoteReader = async (tr, keys) => {
      if (tr.equals(failing)) throw new Error("read exploded");
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
    const reader = new Reader({ cache, readRemote });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const [ok, bad] = await Promise.allSettled([
      reader.read(tr, 1),
      reader.read(failing, 2),
    ]);
    expect(ok.status).toEqual("fulfilled");
    expect(bad.status).toEqual("rejected");
    if (bad.status === "rejected") expect(bad.reason.message).toEqual("read exploded");
    cache.close();
  });

  it("should reject pending reads when closed", async () => {
    const cache = new Cache();
    const reader = new Reader({
      cache,
      readRemote: basicRemoteReadFunc(vi.fn()),
      batchDebounce: TimeSpan.seconds(10),
    });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const pending = reader.read(tr, 1);
    await new Promise((r) => setTimeout(r, 5));
    await reader.close();
    await expect(pending).rejects.toThrow(UnexpectedError);
    cache.close();
  });

  it("should reject reads issued after close", async () => {
    const cache = new Cache();
    const reader = new Reader({ cache, readRemote: basicRemoteReadFunc(vi.fn()) });
    await reader.close();
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    await expect(reader.read(tr, 1)).rejects.toThrow(UnexpectedError);
    cache.close();
  });
});

describe("hung fetch", () => {
  const HUNG_TR = new TimeRange(TimeSpan.seconds(10), TimeSpan.seconds(12));
  const OK_TR = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));

  // Hangs forever for reads of HUNG_TR, resolves immediately otherwise.
  const hangingRemote =
    (fn: Mock): RemoteReader =>
    async (tr, keys) => {
      fn(tr, keys);
      if (tr.equals(HUNG_TR)) return await new Promise<never>(() => {});
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

  it("should resolve a same-batch read whose own fetch succeeded", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: hangingRemote(remoteReadF),
      batchDebounce: TimeSpan.milliseconds(20),
      fetchTimeout: TimeSpan.milliseconds(250),
    });
    const hung = reader.read(HUNG_TR, 1);
    const ok = reader.read(OK_TR, 2);
    const res = await ok;
    expect(res).toHaveLength(3);
    expect(remoteReadF).toHaveBeenCalledTimes(2);
    await expect(hung).rejects.toThrow("timed out after 250ms");
    cache.close();
  });

  it("should serve a later batch while a fetch hangs", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({
      cache,
      readRemote: hangingRemote(remoteReadF),
      batchDebounce: TimeSpan.milliseconds(20),
      fetchTimeout: TimeSpan.milliseconds(500),
    });
    const hung = reader.read(HUNG_TR, 1);
    await new Promise((r) => setTimeout(r, 50));
    expect(remoteReadF).toHaveBeenCalledTimes(1);
    const res = await reader.read(OK_TR, 2);
    expect(res).toHaveLength(3);
    expect(remoteReadF).toHaveBeenCalledTimes(2);
    await reader.close();
    await expect(hung).rejects.toThrow(UnexpectedError);
    cache.close();
  });

  it("should close promptly and reject the hung read", async () => {
    const cache = new Cache();
    let release: ((f: Frame) => void) | undefined;
    const readRemote: RemoteReader = async () =>
      await new Promise<Frame>((r) => (release = r));
    const reader = new Reader({
      cache,
      readRemote,
      batchDebounce: TimeSpan.milliseconds(10),
      fetchTimeout: TimeSpan.seconds(1),
    });
    const hung = reader.read(HUNG_TR, 1);
    await new Promise((r) => setTimeout(r, 30));
    await reader.close();
    await expect(hung).rejects.toThrow(UnexpectedError);
    // A late arrival is discarded without touching the cache.
    release?.(
      new Frame(
        [1],
        [
          new Series({
            data: new Float32Array([1]),
            alignment: 0n,
            timeRange: HUNG_TR,
          }),
        ],
      ),
    );
    await new Promise((r) => setTimeout(r, 10));
    expect(cache.get(1).read(HUNG_TR).gaps).toHaveLength(1);
    cache.close();
  });

  it("should reject a hung read with Unreachable after the fetch timeout", async () => {
    const cache = new Cache();
    const reader = new Reader({
      cache,
      readRemote: hangingRemote(vi.fn()),
      batchDebounce: TimeSpan.milliseconds(10),
      fetchTimeout: TimeSpan.milliseconds(50),
    });
    const hung = reader.read(HUNG_TR, 1);
    await expect(hung).rejects.toSatisfy((e) => Unreachable.matches(e));
    await expect(hung).rejects.toThrow(
      `gap read for ${HUNG_TR.toString()} timed out after 50ms`,
    );
    cache.close();
  });
});

describe("concurrent batches", () => {
  // Batches no longer serialize, so two batches may fetch the same still-open gap.
  // The cache discards the duplicate write and both reads must resolve.
  it("should resolve both reads when two batches fetch the same range", async () => {
    const cache = new Cache();
    const calls: TimeRange[] = [];
    const resolvers: Array<(f: Frame) => void> = [];
    const readRemote: RemoteReader = async (tr) => {
      calls.push(tr);
      return await new Promise<Frame>((r) => resolvers.push(r));
    };
    const reader = new Reader({
      cache,
      readRemote,
      batchDebounce: TimeSpan.milliseconds(10),
    });
    const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3));
    const first = reader.read(tr, 1);
    await new Promise((r) => setTimeout(r, 30));
    const second = reader.read(tr, 1);
    await new Promise((r) => setTimeout(r, 30));
    expect(calls).toEqual([tr, tr]);
    resolvers.forEach((resolve) =>
      resolve(
        new Frame(
          [1],
          [
            new Series({
              data: new Float32Array([1, 2, 3]),
              alignment: 0n,
              timeRange: tr,
            }),
          ],
        ),
      ),
    );
    const [a, b] = await Promise.all([first, second]);
    expect(a).toHaveLength(3);
    expect(b).toHaveLength(3);
    cache.close();
  });
});

describe("read spanning multiple fetches", () => {
  const tr = new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(8));
  const cached = new TimeRange(TimeSpan.seconds(3), TimeSpan.seconds(5));

  const createMiddleEntry = (cache: Cache): void =>
    cache.get(1).writeStatic(
      new MultiSeries([
        new Series({
          data: new Float32Array([1, 2, 3]),
          alignment: 10n,
          timeRange: cached,
        }),
      ]),
    );

  it("should resolve only after every fetch serving the read lands", async () => {
    const cache = new Cache();
    const remoteReadF = vi.fn();
    const reader = new Reader({ cache, readRemote: basicRemoteReadFunc(remoteReadF) });
    createMiddleEntry(cache);
    await reader.read(tr, 1);
    expect(remoteReadF).toHaveBeenCalledTimes(2);
    expect(remoteReadF).toHaveBeenCalledWith(
      new TimeRange(TimeSpan.seconds(1), TimeSpan.seconds(3)),
      [1],
    );
    expect(remoteReadF).toHaveBeenCalledWith(
      new TimeRange(TimeSpan.seconds(5), TimeSpan.seconds(8)),
      [1],
    );
    cache.close();
  });

  it("should reject the read when any of its fetches fails", async () => {
    const cache = new Cache();
    const failing = new TimeRange(TimeSpan.seconds(5), TimeSpan.seconds(8));
    const readRemote: RemoteReader = async (tr, keys) => {
      if (tr.equals(failing)) throw new Error("read exploded");
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
    const reader = new Reader({ cache, readRemote });
    createMiddleEntry(cache);
    await expect(reader.read(tr, 1)).rejects.toThrow("read exploded");
    cache.close();
  });
});
