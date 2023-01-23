// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ChannelPayload } from "./payload";
import { ChannelRetriever } from "./retriever";

export class ChannelRegistry {
  private readonly retriever: ChannelRetriever;
  private readonly channels: Map<string, ChannelPayload>;

  constructor(retriever: ChannelRetriever) {
    this.retriever = retriever;
    this.channels = new Map();
  }

  async get(key: string): Promise<ChannelPayload> {
    let channel = this.channels.get(key);
    if (channel === undefined) {
      channel = await this.retriever.retrieve({ key });
      this.channels.set(key, channel);
    }
    return channel;
  }

  async getN(...keys: string[]): Promise<ChannelPayload[]> {
    const results: ChannelPayload[] = [];
    const retrieveKeys: string[] = [];
    keys.forEach((key) => {
      const channel = this.channels.get(key);
      if (channel === undefined) retrieveKeys.push(key);
      else results.push(channel);
    });
    if (retrieveKeys.length > 0) {
      const channels = await this.retriever.filter({ keys: retrieveKeys });
      channels.forEach((channel) => {
        this.channels.set(channel.key, channel);
        results.push(channel);
      });
    }
    return results;
  }
}
