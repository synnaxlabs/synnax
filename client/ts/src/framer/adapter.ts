// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ChannelKey, ChannelName, ChannelParams } from "@/channel/payload";
import { ChannelRetriever, analyzeChannelParams } from "@/channel/retriever";
import { Frame } from "@/framer/frame";

export class BackwardFrameAdapter {
  private adapter: Map<ChannelKey, ChannelName> | null;
  retriever: ChannelRetriever;
  keys: ChannelKey[];

  private constructor(retriever: ChannelRetriever) {
    this.retriever = retriever;
    this.adapter = null;
    this.keys = [];
  }

  static async open(
    retriever: ChannelRetriever,
    channels: ChannelParams
  ): Promise<BackwardFrameAdapter> {
    const adapter = new BackwardFrameAdapter(retriever);
    await adapter.update(channels);
    return adapter;
  }

  async update(channels: ChannelParams): Promise<void> {
    const { variant, normalized } = analyzeChannelParams(channels);
    if (variant === "keys") {
      this.adapter = null;
      this.keys = normalized;
      return;
    }
    const fetched = await this.retriever.retrieve(normalized);
    this.adapter = new Map<ChannelKey, ChannelName>();
    normalized.forEach((name) => {
      const channel = fetched.find((channel) => channel.name === name);
      if (channel == null) throw new Error(`Channel ${name} not found`);
      // @ts-expect-error;
      this.adapter.set(channel.key, channel.name);
    });
    this.keys = Array.from(this.adapter.keys());
  }

  adapt(fr: Frame): Frame {
    if (this.adapter == null) return fr;
    return fr.map((k, arr) => {
      if (typeof k === "number") {
        // @ts-expect-error
        const name = this.adapter.get(k);
        if (name == null) throw new Error(`Channel ${k} not found`);
        return [name, arr];
      }
      return [k, arr];
    });
  }
}

export class ForwardFrameAdapter {
  private adapter: Map<ChannelName, ChannelKey> | null;
  retriever: ChannelRetriever;
  keys: ChannelKey[];

  private constructor(retriever: ChannelRetriever) {
    this.retriever = retriever;
    this.adapter = null;
    this.keys = [];
  }

  static async open(
    retriever: ChannelRetriever,
    channels: ChannelParams
  ): Promise<ForwardFrameAdapter> {
    const adapter = new ForwardFrameAdapter(retriever);
    await adapter.update(channels);
    return adapter;
  }

  async update(channels: ChannelParams): Promise<void> {
    const { variant, normalized } = analyzeChannelParams(channels);
    if (variant === "keys") {
      this.adapter = null;
      this.keys = normalized;
      return;
    }
    const fetched = await this.retriever.retrieve(normalized);
    this.adapter = new Map<ChannelName, ChannelKey>();
    normalized.forEach((name) => {
      const channel = fetched.find((channel) => channel.name === name);
      if (channel == null) throw new Error(`Channel ${name} not found`);
      // @ts-expect-error;
      this.adapter.set(channel.name, channel.key);
    });
    this.keys = fetched.map((c) => c.key);
  }

  adapt(fr: Frame): Frame {
    if (this.adapter == null) return fr;
    return fr.map((k, arr) => {
      if (typeof k === "string") {
        // @ts-expect-error
        const key = this.adapter.get(k);
        if (key == null) throw new Error(`Channel ${k} not found`);
        return [key, arr];
      }
      return [k, arr];
    });
  }
}
