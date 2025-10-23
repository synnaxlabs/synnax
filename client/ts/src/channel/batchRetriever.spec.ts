// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, errors, Rate } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { channel } from "@/channel";

class MockRetriever implements channel.Retriever {
  func: (
    channels: channel.Params,
    options?: channel.RetrieveOptions,
  ) => Promise<channel.Payload[]>;

  constructor(
    func: (
      channels: channel.Params,
      options?: channel.RetrieveOptions,
    ) => Promise<channel.Payload[]>,
  ) {
    this.func = func;
  }

  async retrieve(
    channels: channel.Params | channel.RetrieveRequest,
    options?: channel.RetrieveOptions,
  ): Promise<channel.Payload[]> {
    if (typeof channels === "object" && !Array.isArray(channels))
      throw new errors.NotImplemented();
    return await this.func(channels, options);
  }
}

describe("channelchannel.Retriever", () => {
  it("should batch multiple retrieve requests", async () => {
    const called = vi.fn();
    const base = new MockRetriever(async (batch): Promise<channel.Payload[]> => {
      called(batch);
      const { normalized } = channel.analyzeParams(batch);
      return normalized.map((key) => ({
        key: key as number,
        name: `channel-${key}`,
        dataType: DataType.FLOAT32,
        internal: false,
        isIndex: false,
        rate: Rate.hz(1),
        leaseholder: 1,
        index: 0,
        virtual: false,
        expression: "",
      }));
    });
    const retriever = new channel.DebouncedBatchRetriever(base, 10);
    const res = await Promise.all([
      retriever.retrieve([1]),
      retriever.retrieve([2]),
      retriever.retrieve([3, 4]),
    ]);
    expect(called).toHaveBeenCalledTimes(1);
    expect(called).toHaveBeenCalledWith([1, 2, 3, 4]);
    expect(res.map((r) => r.map((c) => c.key))).toEqual([[1], [2], [3, 4]]);
  });
  it("should only fetch duplicate keys once", async () => {
    const called = vi.fn();
    const base = new MockRetriever(async (batch): Promise<channel.Payload[]> => {
      called(batch);
      const { normalized } = channel.analyzeParams(batch);
      return normalized.map((key) => ({
        key: key as number,
        name: `channel-${key}`,
        dataType: DataType.FLOAT32,
        internal: false,
        isIndex: false,
        rate: Rate.hz(1),
        leaseholder: 1,
        index: 0,
        virtual: false,
        expression: "",
      }));
    });
    const retriever = new channel.DebouncedBatchRetriever(base, 10);
    const res = await Promise.all([
      retriever.retrieve([1]),
      retriever.retrieve([2]),
      retriever.retrieve([1, 2]),
    ]);
    expect(called).toHaveBeenCalledTimes(1);
    expect(called).toHaveBeenCalledWith([1, 2]);
    expect(res.map((r) => r.map((c) => c.key))).toEqual([[1], [2], [1, 2]]);
  });
  it("should throw an error if the fetch fails", async () => {
    const base = new MockRetriever(async (): Promise<channel.Payload[]> => {
      throw new Error("failed to fetch");
    });
    const retriever = new channel.DebouncedBatchRetriever(base, 10);
    await expect(retriever.retrieve([1])).rejects.toThrow("failed to fetch");
  });
});
