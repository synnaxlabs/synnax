// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeStamp } from "@synnaxlabs/client";
import { fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { client, createStoredVideo, renderVideo } from "@/feature/video/testutil";
import { Video } from "@/feature/video/Video";
import { Session } from "@/session";
import { uniqueName } from "@/testutil";

// The view shows video only where WebCodecs exists.
beforeAll(() => vi.stubGlobal("VideoDecoder", class {}));
afterAll(() => vi.unstubAllGlobals());

describe("video/Video", () => {
  it("should offer to select a channel when the view has none", async () => {
    const { wrapper } = await renderVideo(Video);
    fireEvent.click(await screen.findByText("Select a channel"));
    expect(screen.getByText("No video channel")).toBeDefined();
    const { result } = renderHook(() => Session.Nav.useSelectBottomVisible(), {
      wrapper,
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("should play the range of the view", async () => {
    const ch = await createStoredVideo();
    const range = await client.ranges.create({
      name: uniqueName("test"),
      timeRange: new TimeRange(TimeStamp.seconds(10), TimeStamp.seconds(20)),
    });
    await renderVideo(Video, {
      video: { channel: ch.key, range: { variant: "key", key: range.key } },
    });
    expect(await screen.findByLabelText("Play")).toBeDefined();
    expect(screen.getByLabelText("Time")).toBeDefined();
  });

  it("should show live video with no controls when the view has no range", async () => {
    const ch = await createStoredVideo();
    await renderVideo(Video, { video: { channel: ch.key } });
    expect(await screen.findByText("Waiting for video")).toBeDefined();
    expect(screen.queryByLabelText("Play")).toBeNull();
  });
});
