// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { newClient } from "@/setupspecs";

const client = newClient();

describe("Client", () => {
  describe("read + write", () => {
    it("should correctly write and read a frame of data", async () => {
      const rand = `${TimeStamp.now().toString()}${Math.random()}`;
      const time = await client.channels.create({
        name: `time-${rand}`,
        dataType: "timestamp",
        isIndex: true,
      });
      const data = await client.channels.create({
        name: `data-${rand}`,
        dataType: "float32",
        index: time.key,
      });
      const start = TimeStamp.now();
      await client.write(start, { [time.key]: [start], [data.key]: [1] });
      const frame = await client.read({ start, end: start.add(TimeSpan.seconds(1)) }, [
        time.key,
        data.key,
      ]);
      expect(Array.from(frame.get(time.key))).toEqual([start.valueOf()]);
      expect(Array.from(frame.get(data.key))).toEqual([1]);
    });
    it("should correctly write a single series of data", async () => {
      const rand = `${TimeStamp.now().toString()}${Math.random()}`;
      const time = await client.channels.create({
        name: `time-${rand}`,
        dataType: "timestamp",
        isIndex: true,
      });
      const data = await client.channels.create({
        name: `data-${rand}`,
        dataType: "float32",
        index: time.key,
      });
      const start = TimeStamp.now();
      await client.write(start, time.key, TimeStamp.now());
      await client.write(start, data.key, 1);
    });
  });
  describe("retrieveGroup", () => {
    it("should correctly retrieve the main channel group", async () => {
      const group = await client.channels.retrieveGroup();
      expect(group.name).toEqual("Channels");
    });
  });
});
