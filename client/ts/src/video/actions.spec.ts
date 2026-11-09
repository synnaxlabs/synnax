// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ranger } from "@/ranger";
import { video } from "@/video";

const KEY = "11111111-1111-4111-8111-111111111111";

const createEmpty = (overrides: Partial<video.Video> = {}): video.Video =>
  video.videoZ.parse({ key: KEY, name: "Camera", ...overrides });

const apply = (state: video.Video, ...actions: video.Action[]): video.Video =>
  video.reduceAll(state, actions).next;

const roundTrip = (state: video.Video, ...actions: video.Action[]): video.Video => {
  const { next, inverse } = video.reduceAll(state, actions);
  return video.reduceAll(next, inverse).next;
};

const setName = (name: string): video.Action =>
  video.set({ path: ["name"], value: name });

const RANGE: ranger.Ref = ranger.refZ.parse({
  variant: "key",
  key: "22222222-2222-4222-8222-222222222222",
});

describe("video reducer", () => {
  describe("create", () => {
    it("should replace the state with the created view", () => {
      const created = createEmpty({ name: "created", channel: 3 });
      expect(apply(createEmpty(), video.create({ video: created }))).toEqual(created);
    });
  });

  describe("set of the name", () => {
    it("should set the name and round-trip through its inverse", () => {
      const state = createEmpty({ name: "before" });
      expect(apply(state, setName("after")).name).toEqual("after");
      expect(roundTrip(state, setName("after")).name).toEqual("before");
    });
  });

  describe("setChannel", () => {
    it("should set the channel and round-trip through its inverse", () => {
      const state = createEmpty({ channel: 1 });
      expect(apply(state, video.setChannel({ channel: 2 })).channel).toEqual(2);
      expect(roundTrip(state, video.setChannel({ channel: 2 })).channel).toEqual(1);
    });

    it("should do nothing when the channel is the same", () => {
      const result = video.reduceAll(createEmpty({ channel: 1 }), [
        video.setChannel({ channel: 1 }),
      ]);
      expect(result.inverse).toEqual([]);
      expect(result.targets).toEqual([]);
    });
  });

  describe("setPlaybackRange", () => {
    it("should set the playback range and round-trip through its inverse", () => {
      const state = createEmpty();
      const action = video.setPlaybackRange({ playbackRange: RANGE });
      expect(apply(state, action).playbackRange).toEqual(RANGE);
      expect(roundTrip(state, action).playbackRange).toBeUndefined();
    });

    it("should clear the playback range for live video", () => {
      const state = createEmpty({ playbackRange: RANGE });
      expect(apply(state, video.setPlaybackRange({})).playbackRange).toBeUndefined();
    });
  });

  describe("kindOf", () => {
    it("should key one action by its type and a batch as a transaction", () => {
      expect(video.kindOf([])).toEqual("default");
      expect(video.kindOf([setName("x")])).toEqual("set");
      expect(video.kindOf([setName("x"), video.setChannel({ channel: 1 })])).toEqual(
        "transaction",
      );
    });
  });
});
