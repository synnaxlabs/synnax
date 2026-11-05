// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DataType,
  id,
  Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type VideoFrame,
} from "@synnaxlabs/x";
import { afterAll, describe, expect, it, onTestFinished } from "vitest";

import { type channel } from "@/channel";
import { type Snapshot } from "@/framer/subscription";
import { createTestClient } from "@/testutil";

const client = createTestClient();
const feed = client.openFeed();

afterAll(async () => await feed.close());

// T0 starts a tile of level 13, so blocks hold frames [0, 8], [9, 16], [17, 24], ...
const T0 = TimeStamp.milliseconds(8192 * 1000);
const at = (i: number): TimeStamp => T0.add(TimeSpan.seconds(i));

// Creates a video channel with 30 frames, one each second from T0. Frame i holds the
// byte i, and keyframes are the frames at the given positions.
const createVideo = async (keyframes: number[]): Promise<channel.Channel> => {
  const idx = await client.channels.create({
    name: id.create(),
    isIndex: true,
    dataType: DataType.TIMESTAMP,
  });
  const video = await client.channels.create({
    name: id.create(),
    index: idx.key,
    dataType: DataType.VIDEO,
  });
  onTestFinished(async () => await client.channels.delete([video.key, idx.key]));
  const frames: VideoFrame[] = Array.from({ length: 30 }, (_, i) => ({
    keyframe: keyframes.includes(i),
    codec: "h264",
    data: new Uint8Array([i]),
  }));
  await client.write(T0, {
    [idx.key]: Array.from({ length: 30 }, (_, i) => at(i)),
    [video.key]: new Series({ data: frames, dataType: DataType.VIDEO }),
  });
  return video;
};

const settled = async (read: () => Snapshot): Promise<Snapshot> => {
  await expect.poll(() => read().status.variant).not.toBe("loading");
  return read();
};

const framesOf = (s: Snapshot): number[] =>
  s.series.series.flatMap((x) =>
    [...(x as unknown as Iterable<VideoFrame>)].map((f) => f.data[0]),
  );

const timesOf = (s: Snapshot): TimeStamp[] =>
  s.timestamps!.series.flatMap((x) => [...x.data].map((t) => new TimeStamp(t)));

const range = (from: number, to: number): number[] =>
  Array.from({ length: to - from + 1 }, (_, i) => from + i);

describe("frames subscription", () => {
  it("should serve the frames from the keyframe before the window, with their times", async () => {
    const video = await createVideo([0, 10, 20]);
    const sub = feed.subscribe({
      keys: [video.key],
      window: new TimeRange(at(15), at(17.5)),
      limit: { variant: "frames" },
    });
    onTestFinished(() => sub.close());
    const snapshot = await settled(() => sub.value(video.key));
    expect(snapshot.status.variant).toBe("success");
    expect(framesOf(snapshot)).toEqual(range(9, 24));
    expect(timesOf(snapshot)).toEqual(range(9, 24).map(at));
    snapshot.series.series.forEach((s, i) =>
      expect(s.alignment).toBe(snapshot.timestamps!.series[i].alignment),
    );
  });

  it("should serve the frames of a window that moves forward", async () => {
    const video = await createVideo([0, 10, 20]);
    const sub = feed.subscribe({
      keys: [video.key],
      window: new TimeRange(at(15), at(17.5)),
      limit: { variant: "frames" },
    });
    onTestFinished(() => sub.close());
    await settled(() => sub.value(video.key));
    sub.update({
      keys: [video.key],
      window: new TimeRange(at(25), at(27)),
      limit: { variant: "frames" },
    });
    await expect.poll(() => framesOf(sub.value(video.key))).toEqual(range(17, 29));
  });

  it("should start at the window for a channel with no keyframe before it", async () => {
    const video = await createVideo([10, 20]);
    const sub = feed.subscribe({
      keys: [video.key],
      window: new TimeRange(at(2), at(5)),
      limit: { variant: "frames" },
    });
    onTestFinished(() => sub.close());
    expect(framesOf(await settled(() => sub.value(video.key)))).toEqual(range(0, 8));
  });

  it("should fail a channel that is not video", async () => {
    const idx = await client.channels.create({
      name: id.create(),
      isIndex: true,
      dataType: DataType.TIMESTAMP,
    });
    const data = await client.channels.create({
      name: id.create(),
      index: idx.key,
      dataType: DataType.FLOAT32,
    });
    const sub = feed.subscribe({
      keys: [data.key],
      window: new TimeRange(at(0), at(1)),
      limit: { variant: "frames" },
    });
    onTestFinished(() => sub.close());
    const { status } = await settled(() => sub.value(data.key));
    expect(status.variant).toBe("error");
    expect(status.message).toBe("Failed to read channel data");
    expect(status.description).toContain(
      "cannot seek to a keyframe of a float32 channel",
    );
  });

  it("should fail a virtual channel", async () => {
    const virtual = await client.channels.create({
      name: id.create(),
      virtual: true,
      dataType: DataType.VIDEO,
    });
    onTestFinished(async () => await client.channels.delete(virtual.key));
    const sub = feed.subscribe({
      keys: [virtual.key],
      window: new TimeRange(at(0), at(1)),
      limit: { variant: "frames" },
    });
    onTestFinished(() => sub.close());
    const { status } = await settled(() => sub.value(virtual.key));
    expect(status.variant).toBe("error");
    expect(status.description).toBe(
      `virtual channel ${virtual.name} has no samples to play`,
    );
  });
});
