// Copyright 2026 Synnax Labs, Inc.
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

/** How strongly a writer claims a channel. The highest authority holds control. */
export type Authority = control.Authority;
/** The highest authority. A writer holding it cannot be displaced. */
export const ABSOLUTE_AUTHORITY = control.ABSOLUTE_AUTHORITY;
/** The lowest authority. Any other writer displaces it. */
export const ZERO_AUTHORITY = control.ZERO_AUTHORITY;
/** Control of one channel moving between writers, or being released. */
export type Transfer = control.Transfer<typeof channel.keyZ>;
/** Who holds control of a channel, and at what authority. */
export interface State extends control.State<typeof channel.keyZ> {}
/** A named party that can hold control. */
export interface Subject extends control.Subject {}
/** Zod schema for {@link State}. */
export const stateZ = control.stateZ(z.number());

/** Renders a {@link Transfer} as readable text, for logs and status messages. */
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

/**
 * Tracks who holds control of every channel on a Core, updating as control moves. It
 * observes {@link Transfer} batches; close it to release the underlying stream.
 */
export class StateTracker
  extends framer.ObservableStreamer<Transfer[]>
  implements observe.ObservableAsyncCloseable<Transfer[]>
{
  readonly states: Map<channel.Key, State>;
  private readonly codec: binary.JSONCodec;

  constructor(streamer: framer.Streamer) {
    super(streamer, (frame) => {
      const raw = Array.from(frame.series[0].as("string"));
      const updates: Update[] = raw.map((r) => this.codec.decodeString(r, updateZ));
      updates.forEach((u) => this.merge(u));
      return [updates.flatMap((u) => u.transfers), true];
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
