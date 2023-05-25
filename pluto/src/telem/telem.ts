import { Deep } from "@synnaxlabs/x";

import { Client, ReadResponse, StreamHandler } from "./client";
import { ChannelRange } from "./range";

export class VisTelem {
  private ranges: ChannelRange[];
  readonly data: ReadResponse[][];
  private handler: (() => void) | null;
  private readonly streamHandlers: Map<string, StreamHandler>;

  constructor() {
    this.ranges = [];
    this.data = [];
    this.handler = null;
    this.streamHandlers = new Map();
  }

  listen(handler: () => void): void {
    this.handler = handler;
  }

  async update(client: Client, ranges: ChannelRange[]): Promise<void> {
    const oldRanges = this.ranges;
    this.ranges = ranges;

    // Read new ranges and update existing ranges.
    for (let i = 0; i < this.ranges.length; i++) {
      const range = this.ranges[i];
      const or = oldRanges.find((r) => r.key === range.key);
      if (or == null || !Deep.equal(range, or)) {
        const data = await client.read(range);
        this.data[i] = data;

        // Update our handlers for live ranges.
        if (range.variant === "dynamic") {
          let h = this.streamHandlers.get(range.key);
          if (h == null) {
            h = (data) => {
              if (data != null) this.data[i].push(...data);
              this.handler?.();
            };
            this.streamHandlers.set(range.key, h);
          }
          client.setStreamhandler(h, range);
        }
      }
    }

    // Stop listening to any old live ranges.
    const removedLive = oldRanges
      .filter((r) => !this.ranges.some((nr) => nr.key === r.key))
      .filter((r) => r.variant === "dynamic");
    for (const range of removedLive) {
      const h = this.streamHandlers.get(range.key);
      if (h != null) {
        client.removeStreamHandler(h);
        this.streamHandlers.delete(range.key);
      }
    }
  }
}
