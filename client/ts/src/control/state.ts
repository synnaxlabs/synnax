// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary } from "@synnaxlabs/x/binary";
import { type Destructor } from "@synnaxlabs/x/destructor";
import { observe } from "@synnaxlabs/x/observe";
import { z } from "zod";

import { type Key as ChannelKey } from "@/channel/payload";
import { Authority } from "@/control/authority";
import { type Client as FrameClient } from "@/framer/client";
import { type Streamer as FrameStreamer } from "@/framer/streamer";

export const subjectZ = z.object({
  name: z.string(),
  key: z.string(),
});

export interface Subject {
  name: string;
  key: string;
}

export const stateZ = z.object({
  subject: subjectZ,
  resource: z.number(),
  authority: Authority.z,
});

export interface State {
  subject: Subject;
  resource: ChannelKey;
  authority: Authority;
}

export const filterTransfersByChannelKey =
  (...resources: ChannelKey[]) =>
  (transfers: Transfer[]): Transfer[] =>
    transfers.filter((t) => {
      let ok = false;
      if (t.to != null) ok = resources.includes(t.to.resource);
      if (t.from != null && !ok) ok = resources.includes(t.from.resource);
      return ok;
    });

interface Release {
  from: State;
  to?: null;
}

interface Acquire {
  from?: null;
  to: State;
}

export type Transfer =
  | {
      from: State;
      to: State;
    }
  | Release
  | Acquire;

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
  transfers: Transfer[];
}

export class StateTracker implements observe.Observable<Transfer[]> {
  readonly states: Map<ChannelKey, State>;
  private readonly streamer: FrameStreamer;
  private readonly ecd: binary.EncoderDecoder;
  private readonly observer: observe.Observer<Transfer[]>;
  private readonly closePromise: Promise<void>;

  private constructor(streamer: FrameStreamer) {
    this.states = new Map();
    this.ecd = new binary.JSONEncoderDecoder();
    this.observer = new observe.Observer<Transfer[]>();
    this.streamer = streamer;
    this.closePromise = this.stream();
  }

  subjects(): Subject[] {
    const subjects = new Map<string, Subject>();
    this.states.forEach((s) => subjects.set(s.subject.key, s.subject));
    return Array.from(subjects.values());
  }

  onChange(handler: observe.Handler<Transfer[]>): Destructor {
    return this.observer.onChange(handler);
  }

  async close(): Promise<void> {
    this.streamer.close();
    await this.closePromise;
  }

  static async open(client: FrameClient): Promise<StateTracker> {
    const streamer = await client.openStreamer("sy_node_1_control");
    return new StateTracker(streamer);
  }

  private async stream(): Promise<void> {
    for await (const frame of this.streamer) {
      const update: Update = this.ecd.decode(frame.series[0].buffer);
      this.merge(update);
      this.observer.notify(update.transfers);
    }
  }

  private merge(update: Update): void {
    update.transfers.forEach((t) => {
      if (t.from == null && t.to == null) console.warn("Invalid transfer: ", t);
      if (t.to == null) this.states.delete(t.from.resource);
      else this.states.set(t.to.resource, t.to);
    });
  }
}
