// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control } from "@synnaxlabs/x";
import { binary } from "@synnaxlabs/x/binary";
import { type observe } from "@synnaxlabs/x/observe";
import { z } from "zod";

import { type channel } from "@/channel";
import { framer } from "@/framer";

export interface Authority extends control.Authority {}
export const Authority = control.Authority;
export type Transfer = control.Transfer<channel.Key>;
export interface State extends control.State<channel.Key> {}
export interface Subject extends control.Subject {}
export const stateZ = control.stateZ(z.number());

export const transferString = (t: Transfer): string => {
  if (t.to == null) return `${t.from?.resource} - ${t.from?.subject.name} -> released`;
  if (t.from == null)
    return `${t.to.resource} - released -> ${
      t.to.subject.name
    } (${t.to.authority.toString()})`;
  return `${t.to.resource} - ${t.from.subject.name} -> ${
    t.to.subject.name
  } (${t.to.authority.toString()})`;
};

interface Update {
  transfers: control.Transfer<channel.Key>[];
}

export class StateTracker
  extends framer.ObservableStreamer<Transfer[]>
  implements observe.ObservableAsyncCloseable<Transfer[]>
{
  readonly states: Map<channel.Key, State>;
  private readonly codec: binary.Codec;

  constructor(streamer: framer.Streamer) {
    super(streamer, (frame) => {
      const update: Update = this.codec.decode(frame.series[0].buffer);
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
