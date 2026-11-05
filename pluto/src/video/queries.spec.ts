// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, video } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { Video } from "@/video";

const client = createTestClient();

describe("video queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createVideo = async (
    overrides: Partial<video.New> = {},
  ): Promise<video.Video> => {
    const proj = await client.projects.create({ name: `video_${uuid.create()}` });
    return await client.videos.create(proj.key, { name: "camera", ...overrides });
  };

  // Loads the view into the cache, then renders the hook.
  const load = async <T>(key: video.Key, hook: () => T) => {
    const retrieve = await renderHookSuspended(() => Video.use({ key }), { wrapper });
    await waitFor(() => expect(retrieve.result.current).toBeDefined());
    return renderHook(hook, { wrapper });
  };

  it("should read the name and channel of a view", async () => {
    const created = await createVideo({ name: "chamber", channel: 12 });
    const { result } = await load(created.key, () => ({
      name: Video.useName({ key: created.key }),
      channel: Video.useChannel({ key: created.key }),
    }));
    expect(result.current).toEqual({ name: "chamber", channel: 12 });
  });

  it("should dispatch a channel change to the cache and the Core", async () => {
    const created = await createVideo();
    const { result } = await load(created.key, () => ({
      channel: Video.useChannel({ key: created.key }),
      dispatch: Video.useSingleDispatch({ key: created.key }),
    }));
    act(() => result.current.dispatch(video.setChannel({ channel: 7 })));
    await waitFor(() => expect(result.current.channel).toEqual(7));
    await expect
      .poll(async () => (await client.videos.retrieve({ key: created.key })).channel)
      .toEqual(7);
  });

  it("should create a view and cache it", async () => {
    const project = await client.projects.create({ name: `video_${uuid.create()}` });
    const { result } = renderHook(() => Video.useCreate(), { wrapper });
    const key = uuid.create();
    await act(async () => {
      await result.current.updateAsync({ key, project: project.key, name: "created" });
    });
    expect(result.current.variant).toEqual("success");
    expect((await client.videos.retrieve({ key })).name).toEqual("created");
  });

  it("should rename a view", async () => {
    const created = await createVideo();
    const { result } = await load(created.key, () => ({
      name: Video.useName({ key: created.key }),
      rename: Video.useRename(),
    }));
    await act(async () => {
      await result.current.rename.updateAsync({ key: created.key, name: "renamed" });
    });
    await waitFor(() => expect(result.current.name).toEqual("renamed"));
    expect((await client.videos.retrieve({ key: created.key })).name).toEqual(
      "renamed",
    );
  });

  it("should delete a view", async () => {
    const created = await createVideo();
    const { result } = renderHook(() => Video.useDelete(), { wrapper });
    await act(async () => {
      await result.current.updateAsync(created.key);
    });
    expect(result.current.variant).toEqual("success");
    await expect(client.videos.retrieve({ key: created.key })).rejects.toThrow(
      NotFoundError,
    );
  });
});
