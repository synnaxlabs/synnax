// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, framer } from "@synnaxlabs/client";
import { sync } from "@synnaxlabs/x";

import { type FrameHandler } from "@/flux/aether/types";
import { type Status } from "@/status";

interface MutexValue {
  streamer: framer.ObservableStreamer | null;
}

export class Streamer {
  private readonly handlers: Map<FrameHandler, channel.Name> = new Map();
  private readonly streamerMutex: sync.Mutex<MutexValue> = sync.newMutex({
    streamer: null,
  });
  private readonly handleError: Status.ErrorHandler;
  private openStreamer: framer.StreamOpener | null = null;

  constructor(handleError: Status.ErrorHandler) {
    this.handleError = handleError;
  }

  private handleChange(frame: framer.Frame) {
    const namesInFrame = new Set([...frame.uniqueNames]);
    this.handlers.forEach((channel, handler) => {
      if (!namesInFrame.has(channel)) return;
      try {
        handler(frame);
      } catch (e) {
        this.handleError(
          e,
          `Error calling Flux Frame Handler on channel(s): ${channel}`,
        );
      }
    });
  }

  async close() {
    return await this.streamerMutex.runExclusive(this.unprotectedClose.bind(this));
  }

  private async unprotectedClose() {
    const streamer = this.streamerMutex.streamer;
    this.streamerMutex.streamer = null;
    if (streamer != null) await streamer.close();
    this.handlers.clear();
    this.openStreamer = null;
  }

  async updateStreamer(streamOpener?: framer.StreamOpener): Promise<void> {
    await this.streamerMutex.runExclusive(async () => {
      if (streamOpener != null) this.openStreamer = streamOpener;
      if (this.openStreamer == null) return;
      const names = new Set(this.handlers.values());
      const streamer = this.streamerMutex.streamer;
      if (streamer != null) {
        if (names.size === 0) return await this.unprotectedClose();

        return await streamer.update(Array.from(names));
      }
      if (names.size === 0) return;
      const hardenedStreamer = await framer.HardenedStreamer.open(
        this.openStreamer,
        Array.from(names),
      );
      this.streamerMutex.streamer = new framer.ObservableStreamer(hardenedStreamer);
      this.streamerMutex.streamer.onChange(this.handleChange.bind(this));
    });
  }

  addListener(handler: FrameHandler, channel: channel.Name, onOpen?: () => void) {
    const prevNames = new Set(this.handlers.values());
    this.handlers.set(handler, channel);
    if (!prevNames.has(channel))
      this.handleError(async () => {
        await this.updateStreamer();
        onOpen?.();
      }, `Failed to add ${channel} to the flux streamer`);
    return () => {
      this.handlers.delete(handler);
      this.handleError(
        this.updateStreamer.bind(this),
        `Failed to remove ${channel} from the flux streamer`,
      );
    };
  }
}
