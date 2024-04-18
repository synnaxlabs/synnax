import { describe, expect, it, vi } from "vitest";
import { newClient } from "@/setupspecs";
import { DebouncedBatchRetriever, Retriever, analyzeParams } from "@/channel/retriever";
import { Params, Payload } from "@/channel/payload";
import { DataType, Rate } from "@synnaxlabs/x";



class MockRetriever implements Retriever {
  func: (channels: Params, rangeKey?: string) => Promise<Payload[]>;

  constructor(func: (channels: Params, rangeKey?: string) => Promise<Payload[]>) {
    this.func = func;
  }

  async search(term: string, rangeKey?: string): Promise<Payload[]> {
    throw new Error("Method not implemented.");
  }

  async page(offset: number, limit: number, rangeKey?: string): Promise<Payload[]> {
    throw new Error("Method not implemented.");
  }

  async retrieve(channels: Params, rangeKey?: string): Promise<Payload[]> {
    return this.func(channels, rangeKey);
  }

}


describe("channelRetriever", () => {
  it("should batch multiple retrieve requests", async () => {
    const called = vi.fn();
    const base = new MockRetriever(async (batch): Promise<Payload[]> => {
      called(batch);
      const {normalized} = analyzeParams(batch);
      return normalized.map(
        (key) =>
          ({
            key: key as number,
            name: `channel-${key}`,
            dataType: DataType.FLOAT32,
            isIndex: false,
            rate: Rate.hz(1),
            leaseholder: 1,
            index:0 
          }),
      );
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
      const {normalized} = analyzeParams(batch);
      return normalized.map(
        (key) =>
          ({
            key: key as number,
            name: `channel-${key}`,
            dataType: DataType.FLOAT32,
            isIndex: false,
            rate: Rate.hz(1),
            leaseholder: 1,
            index:0 
          }),
      );
    })
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
    const base = new MockRetriever(async (batch): Promise<Payload[]> => {
      throw new Error("failed to fetch");
    });
    const retriever = new DebouncedBatchRetriever(base, 10);
    await expect(retriever.retrieve([1])).rejects.toThrow("failed to fetch");
  });
});
