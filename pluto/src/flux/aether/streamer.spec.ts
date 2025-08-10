// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type framer } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Streamer } from "@/flux/aether/streamer";
import { type FrameHandler } from "@/flux/aether/types";
import { type Status } from "@/status";

class MockHardenedStreamer implements framer.Streamer {
  private keysI: channel.Params[];
  readonly updateVi = vi.fn();
  readonly closeVi = vi.fn();
  readonly iteratorVi = vi.fn();
  readonly nextVi = vi.fn();
  readonly reads?: framer.Frame[];
  readonly nextFn?: () => Promise<IteratorResult<framer.Frame>>;

  constructor(
    keys: channel.Keys,
    nextFn?: () => Promise<IteratorResult<framer.Frame>>,
    reads?: framer.Frame[],
  ) {
    this.keysI = [keys];
    this.reads = reads;
    this.nextFn = nextFn;
  }

  get keys(): channel.Keys {
    return this.keysI.at(-1) as channel.Keys;
  }

  update(keys: channel.Params): Promise<void> {
    this.keysI.push(keys);
    this.updateVi(keys);
    return Promise.resolve();
  }

  close(): void {
    this.closeVi();
  }

  async next(): Promise<IteratorResult<framer.Frame>> {
    if (this.reads == null && this.nextFn == null) {
      await new Promise((resolve) => setTimeout(resolve, 1000));
      return { done: true, value: undefined };
    }
    if (this.nextFn != null) return await this.nextFn();
    const fr = this.reads?.shift();
    this.nextVi(fr);
    if (fr == null) return { done: true, value: undefined };

    return { done: false, value: fr };
  }

  async read(): Promise<framer.Frame> {
    const res = await this.next();
    if (res.done) throw new Error("no more frames");
    return res.value;
  }

  [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
    this.iteratorVi();
    return this;
  }
}

const mockHandleError: Status.ErrorHandler = (excOrFunc: unknown, message?: string) => {
  if (typeof excOrFunc === "function")
    void (async () => {
      await excOrFunc();
    })();
  else mockHandleError(excOrFunc, message);
};

describe("Streamer", () => {
  let streamer: Streamer;
  let mockStreamOpener: framer.StreamOpener;
  let mockHardenedStreamer: MockHardenedStreamer;

  beforeEach(() => {
    streamer = new Streamer({ handleError: mockHandleError });
    mockHardenedStreamer = new MockHardenedStreamer([]);
    mockStreamOpener = vi.fn().mockResolvedValue(mockHardenedStreamer);
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  describe("constructor", () => {
    it("should initialize with empty handlers", () => {
      expect(streamer).toBeInstanceOf(Streamer);
    });
  });

  describe("addListener", () => {
    it("should add a basic listener", async () => {
      const handler: FrameHandler = vi.fn();
      const channelName = `test_channel_${id.create()}`;
      const onOpen = vi.fn();
      const cleanup = streamer.addListener(handler, channelName, onOpen);
      await streamer.updateStreamer(mockStreamOpener);
      expect(typeof cleanup).toBe("function");
      expect(mockStreamOpener).toHaveBeenCalledWith({
        channels: [channelName],
        downsampleFactor: 1,
        useExperimentalCodec: true,
      });
      await expect.poll(() => onOpen.mock.calls.length > 0).toBe(true);
    });

    it("should return a cleanup function that removes the listener", async () => {
      const handler: FrameHandler = vi.fn();
      const channelName = `test_channel_${id.create()}`;

      const cleanup = streamer.addListener(handler, channelName);
      await streamer.updateStreamer(mockStreamOpener);

      cleanup();

      await expect
        .poll(() => mockHardenedStreamer.closeVi.mock.calls.length > 0)
        .toBe(true);
    });

    it("should handle multiple listeners for different channels", async () => {
      const handler1: FrameHandler = vi.fn();
      const handler2: FrameHandler = vi.fn();
      const channel1 = `test_channel_1_${id.create()}`;
      const channel2 = `test_channel_2_${id.create()}`;
      streamer.addListener(handler1, channel1);
      streamer.addListener(handler2, channel2);
      await streamer.updateStreamer(mockStreamOpener);
      expect(mockStreamOpener).toHaveBeenCalledWith({
        channels: [channel1, channel2],
        downsampleFactor: 1,
        useExperimentalCodec: true,
      });
    });

    it("should not open duplicate channels", async () => {
      const handler1: FrameHandler = vi.fn();
      const handler2: FrameHandler = vi.fn();
      const channelName = `test_channel_${id.create()}`;
      streamer.addListener(handler1, channelName);
      streamer.addListener(handler2, channelName);
      await streamer.updateStreamer(mockStreamOpener);
      expect(mockStreamOpener).toHaveBeenCalledWith({
        channels: [channelName],
        downsampleFactor: 1,
        useExperimentalCodec: true,
      });
    });
  });

  describe("updateStreamer", () => {
    it("should not create streamer if no handlers are registered", async () => {
      await streamer.updateStreamer(mockStreamOpener);
      expect(mockStreamOpener).not.toHaveBeenCalled();
    });

    it("should create streamer when handlers are registered", async () => {
      const handler: FrameHandler = vi.fn();
      const channelName = "test_channel";
      streamer.addListener(handler, channelName);
      await streamer.updateStreamer(mockStreamOpener);
      expect(mockStreamOpener).toHaveBeenCalledWith({
        channels: [channelName],
        downsampleFactor: 1,
        useExperimentalCodec: true,
      });
    });

    it("should update existing streamer with new channels", async () => {
      const handler1: FrameHandler = vi.fn();
      const handler2: FrameHandler = vi.fn();
      const channel1 = "channel1";
      const channel2 = "channel2";
      streamer.addListener(handler1, channel1);
      await streamer.updateStreamer(mockStreamOpener);
      streamer.addListener(handler2, channel2);
      await streamer.updateStreamer();
      expect(mockHardenedStreamer.updateVi).toHaveBeenCalledWith([channel1, channel2]);
    });

    it("should handle stream opener being null", async () => {
      const handler: FrameHandler = vi.fn();
      const channelName = "test_channel";
      streamer.addListener(handler, channelName);
      await streamer.updateStreamer();
      expect(mockStreamOpener).not.toHaveBeenCalled();
    });

    it("should reuse existing stream opener if not provided", async () => {
      const handler1: FrameHandler = vi.fn();
      const handler2: FrameHandler = vi.fn();
      const channel1 = "channel1";
      const channel2 = "channel2";
      streamer.addListener(handler1, channel1);
      await streamer.updateStreamer(mockStreamOpener);
      streamer.addListener(handler2, channel2);
      await streamer.updateStreamer();
      expect(mockHardenedStreamer.updateVi).toHaveBeenCalledWith([channel1, channel2]);
    });
  });

  describe("concurrent operations", () => {
    it("should handle concurrent listener additions", async () => {
      const handler1: FrameHandler = vi.fn();
      const handler2: FrameHandler = vi.fn();
      const channel1 = "channel1";
      const channel2 = "channel2";

      const promises = [
        (async () => {
          streamer.addListener(handler1, channel1);
          await streamer.updateStreamer(mockStreamOpener);
        })(),
        (async () => {
          streamer.addListener(handler2, channel2);
          await streamer.updateStreamer(mockStreamOpener);
        })(),
      ];

      await Promise.all(promises);

      expect(mockStreamOpener).toHaveBeenCalledOnce();
    });

    it("should handle concurrent updates", async () => {
      const handler: FrameHandler = vi.fn();
      const channelName = "test_channel";

      streamer.addListener(handler, channelName);

      const promises = Array(5)
        .fill(null)
        .map(() => streamer.updateStreamer(mockStreamOpener));

      await Promise.all(promises);

      expect(mockStreamOpener).toHaveBeenCalledTimes(1);
    });
  });

  describe("frame handling", () => {
    it("should route frames to appropriate handlers", async () => {
      const handler1: FrameHandler = vi.fn();
      const handler2: FrameHandler = vi.fn();
      const channel1 = "channel1";
      const channel2 = "channel2";
      const channel3 = "channel3";

      const mockFrame = {
        uniqueNames: [channel1, channel3],
      } as framer.Frame;
      mockHardenedStreamer = new MockHardenedStreamer([], async () => {
        await new Promise((resolve) => setTimeout(resolve, 100));
        return {
          done: false,
          value: mockFrame,
        };
      });
      mockStreamOpener = vi.fn().mockResolvedValue(mockHardenedStreamer) as any;

      streamer.addListener(handler1, channel1);
      streamer.addListener(handler2, channel2);

      await streamer.updateStreamer(mockStreamOpener);

      await expect.poll(() => (handler1 as any).mock.calls.length > 0).toBe(true);
      expect(handler1).toHaveBeenCalledWith(mockFrame);
      expect(handler2).not.toHaveBeenCalled();
    });
  });
});
