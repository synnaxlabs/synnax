// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeStamp } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { ReadAdapter, WriteAdapter } from "@/framer/adapter";
import { Frame } from "@/index";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("WriteFrameAdapter", () => {
  let timeCh: channel.Channel;
  let dataCh: channel.Channel;
  let adapter: WriteAdapter;

  beforeAll(async () => {
    timeCh = await client.channels.create({
      name: `time-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    dataCh = await client.channels.create({
      name: `data-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.FLOAT32,
      index: timeCh.key,
    });

    adapter = await WriteAdapter.open(client.channels.retriever, [
      timeCh.key,
      dataCh.key,
    ]);
  });

  it("should correctly adapt a record of keys to single values", async () => {
    const ts = TimeStamp.now().valueOf();
    const res = await adapter.adapt({ [timeCh.key]: ts, [dataCh.key]: 1 });
    expect(res.columns).toHaveLength(2);
    expect(res.series).toHaveLength(2);
    expect(res.get(timeCh.key)).toHaveLength(1);
    expect(res.get(dataCh.key)).toHaveLength(1);
    expect(res.get(timeCh.key).at(0)).toEqual(ts);
    expect(res.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should correctly adapt a record of names to single values", async () => {
    const ts = TimeStamp.now().valueOf();
    const res2 = await adapter.adapt({ [timeCh.name]: ts, [dataCh.name]: 1 });
    expect(res2.columns).toHaveLength(2);
    expect(res2.series).toHaveLength(2);
    expect(res2.get(timeCh.key)).toHaveLength(1);
    expect(res2.get(dataCh.key)).toHaveLength(1);
    expect(res2.get(timeCh.key).at(0)).toEqual(ts);
    expect(res2.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should correctly adapt a single name to a single series", async () => {
    const res3 = await adapter.adapt(dataCh.name, new Series(1));
    expect(res3.columns).toHaveLength(1);
    expect(res3.series).toHaveLength(1);
    expect(res3.get(dataCh.key)).toHaveLength(1);
    expect(res3.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should correctly adapt multiple names to multiple series", async () => {
    const ts = TimeStamp.now().valueOf();
    const res4 = await adapter.adapt(
      [timeCh.name, dataCh.name],
      [new Series(ts), new Series(1)],
    );
    expect(res4.get(timeCh.key)).toHaveLength(1);
    expect(res4.get(dataCh.key)).toHaveLength(1);
    expect(res4.get(timeCh.key).at(0)).toEqual(ts);
    expect(res4.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should correctly adapt a frame keyed by name", async () => {
    const ts = TimeStamp.now().valueOf();
    const fr = new Frame({
      [timeCh.name]: new Series(ts),
      [dataCh.name]: new Series(1),
    });
    const res = await adapter.adapt(fr);
    expect(res.columns).toHaveLength(2);
    expect(res.series).toHaveLength(2);
    expect(res.get(timeCh.key).at(0)).toEqual(ts);
    expect(res.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should not modify a frame keyed by key", async () => {
    const ts = TimeStamp.now().valueOf();
    const fr = new Frame({ [timeCh.key]: new Series(ts), [dataCh.key]: new Series(1) });
    const res = await adapter.adapt(fr);
    expect(res.columns).toHaveLength(2);
    expect(res.series).toHaveLength(2);
    expect(res.get(timeCh.key).at(0)).toEqual(ts);
    expect(res.get(dataCh.key).at(0)).toEqual(1);
  });

  it("should correctly adapt a map of series", async () => {
    const ts = TimeStamp.now().valueOf();
    const m = new Map();
    m.set(timeCh.key, new Series(ts));
    const res = await adapter.adapt(m);
    expect(res.columns).toHaveLength(1);
    expect(res.series).toHaveLength(1);
    expect(res.get(timeCh.key)).toHaveLength(1);
    expect(res.get(timeCh.key).at(0)).toEqual(ts);
  });

  it("should correctly adapt a name and JSON value", async () => {
    const jsonChannel = await client.channels.create({
      name: `json-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.JSON,
      virtual: true,
    });
    const adapter = await WriteAdapter.open(client.channels.retriever, [
      jsonChannel.key,
    ]);
    const res = await adapter.adapt(jsonChannel.name, [{ dog: "blue" }]);
    expect(res.columns).toHaveLength(1);
    expect(res.series).toHaveLength(1);
    expect(res.get(jsonChannel.key)).toHaveLength(1);
    expect(res.get(jsonChannel.key).at(0)).toEqual({ dog: "blue" });
  });

  it("should correctly adapt a name and a json typed series", async () => {
    const jsonChannel = await client.channels.create({
      name: `json-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.JSON,
      virtual: true,
    });
    const adapter = await WriteAdapter.open(client.channels.retriever, [
      jsonChannel.key,
    ]);
    const res = await adapter.adapt(jsonChannel.name, new Series([{ dog: "blue" }]));
    expect(res.columns).toHaveLength(1);
    expect(res.series).toHaveLength(1);
    expect(res.get(jsonChannel.key)).toHaveLength(1);
    expect(res.get(jsonChannel.key).at(0)).toEqual({ dog: "blue" });
  });

  it("should correctly adapt a numeric value to a BigInt keyed by key", async () => {
    const bigIntCh = await client.channels.create({
      name: `bigint-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.INT64,
      virtual: true,
    });
    const res = await adapter.adapt({
      [bigIntCh.key]: 12,
    });
    expect(res.get(bigIntCh.key).at(0)).toEqual(12n);
  });

  describe("adaptParams", () => {
    it("should correctly adapt generic object keys", async () => {
      const res = await adapter.adaptParams([timeCh.name, dataCh.name]);
      expect(res).toContain(timeCh.key);
      expect(res).toContain(dataCh.key);
    });
  });

  describe("update", () => {
    it("should return false when updating with the same channels", async () => {
      const hasChanged = await adapter.update([timeCh.key, dataCh.key]);
      expect(hasChanged).toBe(false);
    });

    it("should return true when adding a new channel", async () => {
      const newCh = await client.channels.create({
        name: `new-${Math.random()}-${TimeStamp.now().toString()}`,
        dataType: DataType.FLOAT32,
        index: timeCh.key,
      });
      const hasChanged = await adapter.update([timeCh.key, dataCh.key, newCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return true when removing a channel", async () => {
      const hasChanged = await adapter.update([timeCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return true when replacing channels", async () => {
      const newCh = await client.channels.create({
        name: `replacement-${Math.random()}-${TimeStamp.now().toString()}`,
        dataType: DataType.FLOAT32,
        index: timeCh.key,
      });
      const hasChanged = await adapter.update([timeCh.key, newCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return false when updating with same channels in different order", async () => {
      await adapter.update([timeCh.key, dataCh.key]);
      const hasChanged = await adapter.update([dataCh.key, timeCh.key]);
      expect(hasChanged).toBe(false);
    });

    it("should return false when updating with channel names that resolve to same keys", async () => {
      await adapter.update([timeCh.key, dataCh.key]);
      const hasChanged = await adapter.update([timeCh.name, dataCh.name]);
      expect(hasChanged).toBe(false);
    });
  });
});

describe("ReadAdapter", () => {
  let timeCh: channel.Channel;
  let dataCh: channel.Channel;
  let adapter: ReadAdapter;

  beforeAll(async () => {
    timeCh = await client.channels.create({
      name: `read-time-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    dataCh = await client.channels.create({
      name: `read-data-${Math.random()}-${TimeStamp.now().toString()}`,
      dataType: DataType.FLOAT32,
      index: timeCh.key,
    });

    adapter = await ReadAdapter.open(client.channels.retriever, [
      timeCh.key,
      dataCh.key,
    ]);
  });

  describe("update", () => {
    it("should return false when updating with the same channels", async () => {
      const hasChanged = await adapter.update([timeCh.key, dataCh.key]);
      expect(hasChanged).toBe(false);
    });

    it("should return true when adding a new channel", async () => {
      const newCh = await client.channels.create({
        name: `read-new-${Math.random()}-${TimeStamp.now().toString()}`,
        dataType: DataType.FLOAT32,
        index: timeCh.key,
      });
      const hasChanged = await adapter.update([timeCh.key, dataCh.key, newCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return true when removing a channel", async () => {
      const hasChanged = await adapter.update([timeCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return true when replacing channels", async () => {
      const newCh = await client.channels.create({
        name: `read-replacement-${Math.random()}-${TimeStamp.now().toString()}`,
        dataType: DataType.FLOAT32,
        index: timeCh.key,
      });
      const hasChanged = await adapter.update([timeCh.key, newCh.key]);
      expect(hasChanged).toBe(true);
    });

    it("should return false when updating with same channels in different order", async () => {
      await adapter.update([timeCh.key, dataCh.key]);
      const hasChanged = await adapter.update([dataCh.key, timeCh.key]);
      expect(hasChanged).toBe(false);
    });

    it("should return false when updating with channel names that resolve to same keys", async () => {
      await adapter.update([timeCh.key, dataCh.key]);
      const hasChanged = await adapter.update([timeCh.name, dataCh.name]);
      expect(hasChanged).toBe(false);
    });
  });
});
