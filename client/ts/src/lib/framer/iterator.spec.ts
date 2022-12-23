import { describe, test, expect } from "vitest";

import { newClient } from "../../setupspecs";
import { Channel } from "../channel";
import { DataType, Rate, TimeRange, TimeSpan, TimeStamp } from "../telem";
import { randomTypedArray } from "../util/telem";

const client = newClient();

const newChannel = async (): Promise<Channel> => {
  return await client.channel.create({
    name: "test",
    nodeId: 1,
    rate: Rate.Hz(25),
    dataType: DataType.Float64,
  });
};

describe("Iterator", () => {
  test("basic iteration", async () => {
    const ch = await newChannel();
    const writer = await client.data.newWriter(TimeStamp.Second, [ch.key]);
    const data = randomTypedArray(25, ch.dataType);
    try {
      await writer.write({ [ch.key]: data });
      await writer.write({ [ch.key]: data });
      await writer.write({ [ch.key]: data });
    } finally {
      await writer.commit();
      await writer.close();
    }
    const iterator = await client.data.newIterator(
      new TimeRange(TimeSpan.Zero, TimeSpan.Seconds(4)),
      [ch.key],
      false
    );
    try {
      expect(await iterator.seekFirst()).toBeTruthy();
      let c = 0;
      while (await iterator.next(TimeSpan.Seconds(1))) {
        c++;
        expect((await iterator.value())[ch.key].data).toHaveLength(25 * 8);
      }
      expect(c).toEqual(3);
    } finally {
      await iterator.close();
    }
  });
});
