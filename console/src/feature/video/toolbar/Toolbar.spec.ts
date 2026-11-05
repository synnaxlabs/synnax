// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeRange, TimeStamp } from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { type VideoFrame } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { client, createStoredVideo, renderVideo } from "@/feature/video/testutil";
import { Toolbar } from "@/feature/video/toolbar";
import {
  captureBrowserDownloads,
  findDialogTriggerByText,
  isPlutoDisabled,
  uniqueName,
} from "@/testutil";

const roles = new RoleClients(client);

// An H.264 keyframe with a 1920x1080 baseline SPS, which an MP4 init segment can
// parse.
const KEYFRAME: VideoFrame = {
  keyframe: true,
  codec: "h264",
  data: new Uint8Array(
    [
      [
        0x67, 0x42, 0xc0, 0x28, 0xd9, 0x00, 0x78, 0x02, 0x27, 0xe5, 0x84, 0x00, 0x00,
        0x03, 0x00, 0x04, 0x00, 0x00, 0x03, 0x00, 0xf0, 0x3c, 0x60, 0xc9, 0x20,
      ],
      [0x68, 0xce, 0x3c, 0x80],
      [0x65, 0x88, 0x84, 0x00],
    ].flatMap((nalu) => [0, 0, 0, 1, ...nalu]),
  ),
};

const createVideoChannel = async () =>
  await client.channels.create({
    name: uniqueName("camera"),
    dataType: DataType.VIDEO,
    virtual: true,
  });

describe("video/toolbar/Toolbar", () => {
  it("should show the name of the view", async () => {
    await renderVideo(Toolbar, { video: { name: "Chamber camera" } });
    expect(await screen.findByText("Chamber camera")).toBeDefined();
  });

  it("should set the channel of the view through the Core", async () => {
    const ch = await createVideoChannel();
    const { key } = await renderVideo(Toolbar);
    fireEvent.click(await findDialogTriggerByText("Select channel"));
    fireEvent.change(await screen.findByPlaceholderText("Search channels..."), {
      target: { value: ch.name },
    });
    fireEvent.click(await screen.findByText(ch.name));
    await expect
      .poll(async () => (await client.videos.retrieve({ key })).channel)
      .toEqual(ch.key);
  });

  it("should show the channel of the view", async () => {
    const ch = await createVideoChannel();
    await renderVideo(Toolbar, { video: { channel: ch.key } });
    expect(await screen.findByText(ch.name)).toBeDefined();
  });

  it("should set the range of the view through the Core", async () => {
    const range = await client.ranges.create({
      name: uniqueName("test"),
      timeRange: new TimeRange(TimeStamp.seconds(10), TimeStamp.seconds(20)),
    });
    const { key } = await renderVideo(Toolbar);
    fireEvent.click(await findDialogTriggerByText("Live"));
    fireEvent.change(await screen.findByPlaceholderText("Search ranges..."), {
      target: { value: range.name },
    });
    fireEvent.click(await screen.findByText(range.name));
    await expect
      .poll(async () => (await client.videos.retrieve({ key })).range)
      .toEqual({ variant: "key", key: range.key });
  });

  it("should show the range of the view", async () => {
    const range = await client.ranges.create({
      name: uniqueName("test"),
      timeRange: new TimeRange(TimeStamp.seconds(10), TimeStamp.seconds(20)),
    });
    await renderVideo(Toolbar, {
      video: { range: { variant: "key", key: range.key } },
    });
    expect(await screen.findByText(range.name)).toBeDefined();
  });

  it("should download the range of the view as MP4", async () => {
    const downloads = captureBrowserDownloads();
    onTestFinished(() => {
      vi.restoreAllMocks();
    });
    const ch = await createStoredVideo();
    await client.write(TimeStamp.seconds(10), {
      [ch.index]: [TimeStamp.seconds(10)],
      [ch.key]: new Series({ data: [KEYFRAME], dataType: DataType.VIDEO }),
    });
    const range = await client.ranges.create({
      name: uniqueName("test"),
      timeRange: new TimeRange(TimeStamp.seconds(5), TimeStamp.seconds(20)),
    });
    await renderVideo(Toolbar, {
      video: { channel: ch.key, range: { variant: "key", key: range.key } },
    });
    fireEvent.click(await screen.findByLabelText("Download as MP4"));
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe("Camera.mp4");
    const file = new TextDecoder("latin1").decode(
      await downloads.blobs[0].arrayBuffer(),
    );
    expect(file.slice(4, 8)).toBe("ftyp");
  });

  it("should not download a view with no range", async () => {
    const ch = await createStoredVideo();
    await renderVideo(Toolbar, { video: { channel: ch.key } });
    expect(
      isPlutoDisabled(await screen.findByLabelText("Set a range to download")),
    ).toBe(true);
  });

  it("should not let a viewer change the channel", async () => {
    await renderVideo(Toolbar, { as: await roles.get("Viewer") });
    await waitFor(() =>
      expect(isPlutoDisabled(screen.getByRole("button", { name: "Channel" }))).toBe(
        true,
      ),
    );
  });
});
