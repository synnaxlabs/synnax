// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError } from "@/index";
import { query } from "@/query";
import { createTestClient, expectDeleted } from "@/testutil";
import { video } from "@/video";

const client = createTestClient();

const createProject = async () =>
  await client.projects.create({ name: `video-${id.create()}` });

describe("Video", () => {
  it("should create a live video view", async () => {
    const proj = await createProject();
    const v = await client.videos.create(proj.key, { name: "Camera", channel: 4 });
    expect(v.name).toEqual("Camera");
    expect(v.key).not.toEqual(uuid.ZERO);
    expect(v.channel).toEqual(4);
    expect(v.range).toBeUndefined();
    expect(await client.videos.retrieve({ key: v.key })).toEqual(v);
  });

  it("should rename a video view", async () => {
    const proj = await createProject();
    const v = await client.videos.create(proj.key, { name: "Camera" });
    await client.videos.rename(v.key, "Chamber");
    await expect
      .poll(async () => (await client.videos.retrieve({ key: v.key })).name)
      .toEqual("Chamber");
  });

  it("should dispatch a channel change to the Core", async () => {
    const proj = await createProject();
    const v = await client.videos.create(proj.key, { name: "Camera" });
    expect(await client.videos.dispatch(v.key, video.setChannel({ channel: 8 }))).toBe(
      true,
    );
    await expect
      .poll(async () => {
        const [fetched] = await client.videos.retrieve([v.key]);
        return fetched.channel;
      })
      .toEqual(8);
  });

  it("should delete a video view", async () => {
    const proj = await createProject();
    const v = await client.videos.create(proj.key, { name: "Camera" });
    await client.videos.delete(v.key);
    await expect(client.videos.retrieve({ key: v.key })).rejects.toThrow(NotFoundError);
  });

  it("should tombstone a delete from the live delete signal", async () => {
    await client.connect();
    const proj = await createProject();
    const v = await client.videos.create(proj.key, { name: `video-${id.create()}` });
    await client.videos.delete(v.key);
    await expect
      .poll(() => query.Deleted.matches(client.videos.getCached(v.key)))
      .toBe(true);
    expect(expectDeleted(client.videos.getCached(v.key)).corpse.name).toEqual(v.name);
  });
});
