// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor, binary, observe } from "@synnaxlabs/x";

import { type Key as ChannelKey } from "@/channel/payload";
import { type Authority } from "@/control/authority";
import { type Client as FrameClient } from "@/framer/client";
import { type Streamer as FrameStreamer } from "@/framer/streamer";

export interface State {
  subject: string;
  resource: ChannelKey;
  authority: Authority;
}

export type Transfer =
  | {
      from: State;
      to: State;
    }
  | {
      from?: State;
      to: State;
    }
  | {
      from: State;
      to?: State;
    };

interface Update {
  transfers: Transfer[];
}

export class StateTracker implements observe.Observable<Transfer[]> {
  readonly states: Map<ChannelKey, State>;
  private readonly streamer: FrameStreamer;
  private readonly ecd: binary.EncoderDecoder;
  private readonly observer: observe.Observer<Transfer[]>;

  private constructor(streamer: FrameStreamer) {
    this.states = new Map();
    this.ecd = new binary.JSONEncoderDecoder();
    this.observer = new observe.Observer<Transfer[]>();
    this.streamer = streamer;
    void this.stream();
  }

  onChange(handler: observe.Handler<Transfer[]>): Destructor {
    return this.observer.onChange(handler);
  }

  close(): void {
    this.streamer.close();
  }

  static async open(client: FrameClient): Promise<StateTracker> {
    const streamer = await client.newStreamer("sy_node_1_control");
    return new StateTracker(streamer);
  }

  private async stream(): Promise<void> {
    for await (const frame of this.streamer) {
      const update: Update = this.ecd.decode(frame.series[0].buffer);
      console.log("CONTROL UPDATE", this.observer, update);
      this.merge(update);
      this.observer.notify(update.transfers);
    }
  }

  private merge(update: Update): void {
    update.transfers.forEach((t) => {
      if (t.to == null) {
        this.states.delete((t.from as State).resource);
      } else {
        this.states.set(t.to.resource, t.to);
      }
    });
  }
}
