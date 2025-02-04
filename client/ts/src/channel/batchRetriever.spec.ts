// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Rate } from "@synnaxlabs/x/telem";
import { describe, expect, it, vi } from "vitest";

import { type Params, type Payload } from "@/channel/payload";
import {
  analyzeChannelParams,
  DebouncedBatchRetriever,
  type RetrieveOptions,
  type Retriever,
} from "@/channel/retriever";

class MockRetriever implements Retriever {
  func: (channels: Params, options?: RetrieveOptions) => Promise<Payload[]>;

  constructor(
    func: (channels: Params, options?: RetrieveOptions) => Promise<Payload[]>,
  ) {
    this.func = func;
  }

  async search(): Promise<Payload[]> {
    throw new Error("Method not implemented.");
  }

  async page(): Promise<Payload[]> {
    throw new Error("Method not implemented.");
  }

  async retrieve(channels: Params, options?: RetrieveOptions): Promise<Payload[]> {
    return await this.func(channels, options);
  }
}

describe("channelRetriever", () => {
  it("should batch multiple retrieve requests", async () => {
    const called = vi.fn();
    const base = new MockRetriever(async (batch): Promise<Payload[]> => {
      called(batch);
      const { normalized } = analyzeChannelParams(batch);
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
        requires: [],
      }));
    });
    const retriever = new DebouncedBatchRetriever(base, 10);
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
    const base = new MockRetriever(async (batch): Promise<Payload[]> => {
      called(batch);
      const { normalized } = analyzeChannelParams(batch);
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
        requires: [],
      }));
    });
    const retriever = new DebouncedBatchRetriever(base, 10);
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
    const base = new MockRetriever(async (): Promise<Payload[]> => {
      throw new Error("failed to fetch");
    });
    const retriever = new DebouncedBatchRetriever(base, 10);
    await expect(retriever.retrieve([1])).rejects.toThrow("failed to fetch");
  });
});
