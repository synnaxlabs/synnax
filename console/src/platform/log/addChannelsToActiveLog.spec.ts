// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType, type log, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { beforeAll, describe, expect, it } from "vitest";

import { addChannelsToActiveLog } from "@/platform/log/addChannelsToActiveLog";
import { uniqueName } from "@/testutil";

const client: Synnax = createTestClient();

const newChannel = async (): Promise<channel.Key> =>
  (
    await client.channels.create({
      name: uniqueName("ch"),
      dataType: DataType.FLOAT32,
      virtual: true,
    })
  ).key;

const newLog = async (project: string): Promise<log.Key> =>
  (await client.logs.create(project, { name: uniqueName("log") })).key;

const channelsOf = async (key: log.Key): Promise<channel.Key[]> =>
  (await client.logs.retrieve(key)).channels.map((e) => e.channel);

describe("addChannelsToActiveLog", () => {
  let project: string;

  beforeAll(async () => {
    project = (await client.projects.create({ name: uniqueName("proj"), layout: {} }))
      .key;
  });

  it("should add a channel that the log does not have", async () => {
    const key = await newLog(project);
    const ch = await newChannel();
    await addChannelsToActiveLog(client, key, [ch]);
    expect(await channelsOf(key)).toEqual([ch]);
  });

  it("should add every channel when the log is empty", async () => {
    const key = await newLog(project);
    const [a, b] = [await newChannel(), await newChannel()];
    await addChannelsToActiveLog(client, key, [a, b]);
    expect(await channelsOf(key)).toEqual(expect.arrayContaining([a, b]));
    expect(await channelsOf(key)).toHaveLength(2);
  });

  it("should skip a channel the log already has", async () => {
    const key = await newLog(project);
    const [a, b] = [await newChannel(), await newChannel()];
    await addChannelsToActiveLog(client, key, [a]);
    await addChannelsToActiveLog(client, key, [a, b]);
    expect(await channelsOf(key)).toHaveLength(2);
    expect(await channelsOf(key)).toEqual(expect.arrayContaining([a, b]));
  });

  it("should leave the log untouched when every channel is already present", async () => {
    const key = await newLog(project);
    const ch = await newChannel();
    await addChannelsToActiveLog(client, key, [ch]);
    await addChannelsToActiveLog(client, key, [ch]);
    expect(await channelsOf(key)).toEqual([ch]);
  });

  it("should do nothing when given no channels", async () => {
    const key = await newLog(project);
    await addChannelsToActiveLog(client, key, []);
    expect(await channelsOf(key)).toHaveLength(0);
  });

  it("should add a repeated channel only once", async () => {
    const key = await newLog(project);
    const ch = await newChannel();
    await addChannelsToActiveLog(client, key, [ch, ch]);
    expect(await channelsOf(key)).toEqual([ch]);
  });
});
