// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key, type Name, type Params } from "@/channel/payload";
import { type Retriever, analyzeParams } from "@/channel/retriever";
import { type Frame } from "@/framer/frame";

export class BackwardFrameAdapter {
  private adapter: Map<Key, Name> | null;
  retriever: Retriever;
  keys: Key[];

  private constructor(retriever: Retriever) {
    this.retriever = retriever;
    this.adapter = null;
    this.keys = [];
  }

  static async open(
    retriever: Retriever,
    channels: Params,
  ): Promise<BackwardFrameAdapter> {
    const adapter = new BackwardFrameAdapter(retriever);
    await adapter.update(channels);
    return adapter;
  }

  async update(channels: Params): Promise<void> {
    const { variant, normalized } = analyzeParams(channels);
    if (variant === "keys") {
      this.adapter = null;
      this.keys = normalized;
      return;
    }
    const fetched = await this.retriever.retrieve(normalized);
    const a = new Map<Key, Name>();
    this.adapter = a;
    normalized.forEach((name) => {
      const channel = fetched.find((channel) => channel.name === name);
      if (channel == null) throw new Error(`Channel ${name} not found`);
      a.set(channel.key, channel.name);
    });
    this.keys = Array.from(this.adapter.keys());
  }

  adapt(fr: Frame): Frame {
    if (this.adapter == null) return fr;
    const a = this.adapter;
    return fr.map((k, arr) => {
      if (typeof k === "number") {
        const name = a.get(k);
        if (name == null) throw new Error(`Channel ${k} not found`);
        return [name, arr];
      }
      return [k, arr];
    });
  }
}

export class ForwardFrameAdapter {
  private adapter: Map<Name, Key> | null;
  retriever: Retriever;
  keys: Key[];

  private constructor(retriever: Retriever) {
    this.retriever = retriever;
    this.adapter = null;
    this.keys = [];
  }

  static async open(
    retriever: Retriever,
    channels: Params,
  ): Promise<ForwardFrameAdapter> {
    const adapter = new ForwardFrameAdapter(retriever);
    await adapter.update(channels);
    return adapter;
  }

  async update(channels: Params): Promise<void> {
    const { variant, normalized } = analyzeParams(channels);
    if (variant === "keys") {
      this.adapter = null;
      this.keys = normalized;
      return;
    }
    const fetched = await this.retriever.retrieve(normalized);
    const a = new Map<Name, Key>();
    this.adapter = a;
    normalized.forEach((name) => {
      const channel = fetched.find((channel) => channel.name === name);
      if (channel == null) throw new Error(`Channel ${name} not found`);
      a.set(channel.name, channel.key);
    });
    this.keys = fetched.map((c) => c.key);
  }

  adapt(fr: Frame): Frame {
    if (this.adapter == null) return fr;
    const a = this.adapter;
    return fr.map((k, arr) => {
      if (typeof k === "string") {
        const key = a.get(k);
        if (key == null) throw new Error(`Channel ${k} not found`);
        return [key, arr];
      }
      return [k, arr];
    });
  }
}
