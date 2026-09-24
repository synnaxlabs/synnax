// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { synnax } from "@/synnax/aether";
import { telem } from "@/telem/aether";

export const stateZ = z.object({
  channel: z.number(),
  time: TimeStamp.z.nullable().optional(),
});

export interface State extends z.input<typeof stateZ> {}

interface InternalState {
  source: telem.NumberSource;
  stopListening: destructor.Destructor;
  seeded: number;
}

export class LatestSample extends aether.Leaf<typeof stateZ, InternalState> {
  static readonly TYPE = "LatestSample";
  static readonly z = stateZ;

  schema = LatestSample.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    const { channel } = this.state;
    i.source = telem.useSource(ctx, telem.streamChannelValue({ channel }), i.source);
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => this.publish(i.source.value()));
    this.publish(i.source.value());
    if (channel === 0 || i.seeded === channel) return;
    i.seeded = channel;
    this.seed(ctx, channel).catch(console.error);
  }

  private async seed(ctx: aether.Context, channel: number): Promise<void> {
    const client = synnax.use(ctx);
    if (client == null) return;
    const last = (await client.readLatest(channel, 1)).at(-1);
    if (this.deleted || channel !== this.state.channel) return;
    if (last == null) this.setState((p) => ({ ...p, time: p.time ?? null }));
    else this.publish(last as bigint);
  }

  private publish(value: number | bigint): void {
    if (typeof value === "number" && Number.isNaN(value)) return;
    const time = new TimeStamp(value);
    if (this.state.time != null && !time.after(this.state.time)) return;
    this.setState((p) => ({ ...p, time }));
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.source.cleanup?.();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [LatestSample.TYPE]: LatestSample,
};
