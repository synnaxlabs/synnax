// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, control, type observe } from "@synnaxlabs/x";
import { z } from "zod";

import { type channel } from "@/channel";
import { keyZ } from "@/channel/types.gen";
import { framer } from "@/framer";

export type Authority = control.Authority;
export const ABSOLUTE_AUTHORITY = control.ABSOLUTE_AUTHORITY;
export const ZERO_AUTHORITY = control.ZERO_AUTHORITY;
export type Transfer = control.Transfer<typeof channel.keyZ>;
export interface State extends control.State<typeof channel.keyZ> {}
export interface Subject extends control.Subject {}
export const stateZ = control.stateZ(z.number());

export const transferString = (t: Transfer): string => {
  const fromResource = t.from?.resource;
  const toResource = t.to?.resource;
  if (t.to == null) return `${fromResource} - ${t.from?.subject.name} -> released`;
  if (t.from == null)
    return `${toResource} - released -> ${
      t.to.subject.name
    } (${t.to.authority.toString()})`;
  return `${toResource} - ${t.from.subject.name} -> ${
    t.to.subject.name
  } (${t.to.authority.toString()})`;
};

const updateZ = z.object({
  transfers: z.array(control.transferZ(keyZ)),
});

export interface Update extends z.infer<typeof updateZ> {}

export class StateTracker
  extends framer.ObservableStreamer<Transfer[]>
  implements observe.ObservableAsyncCloseable<Transfer[]>
{
  readonly states: Map<channel.Key, State>;
  private readonly codec: binary.Codec;

  constructor(streamer: framer.Streamer) {
    super(streamer, (frame) => {
      const update: Update = this.codec.decode(frame.series[0].buffer, updateZ);
      this.merge(update);
      return [update.transfers, true];
    });
    this.states = new Map();
    this.codec = new binary.JSONCodec();
  }

  subjects(): Subject[] {
    const subjects = new Map<string, Subject>();
    this.states.forEach((s) => subjects.set(s.subject.key, s.subject));
    return Array.from(subjects.values());
  }

  private merge(update: Update): void {
    update.transfers.forEach((t) => {
      if (t.from == null && t.to == null) console.warn("Invalid transfer: ", t);
      if (t.to == null) this.states.delete(t.from.resource);
      else this.states.set(t.to.resource, t.to);
    });
  }
}
