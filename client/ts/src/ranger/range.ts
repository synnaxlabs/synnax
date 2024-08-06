// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Series, TimeRange } from "@synnaxlabs/x/telem";

import { type Key, type Name, type Params } from "@/channel/payload";
import { type Retriever as ChannelRetriever } from "@/channel/retriever";
import { QueryError } from "@/errors";
import { type framer } from "@/framer";
import { type label } from "@/label";
import { type Label } from "@/label/payload";
import { ontology } from "@/ontology";
import { type Alias, type Aliaser } from "@/ranger/alias";
import { type KV } from "@/ranger/kv";
import { Payload } from "@/ranger/payload";
import { type signals } from "@/signals";

const ontologyID = (key: string): ontology.ID =>
  new ontology.ID({ type: "range", key });

export class Range {
  key: string;
  name: string;
  readonly kv: KV;
  readonly timeRange: TimeRange;
  readonly color: string | undefined;
  readonly channels: ChannelRetriever;
  private readonly aliaser: Aliaser;
  private readonly frameClient: framer.Client;
  private readonly labelClient: label.Client;

  constructor(
    name: string,
    timeRange: TimeRange = TimeRange.ZERO,
    key: string,
    color: string | undefined,
    _frameClient: framer.Client,
    _kv: KV,
    _aliaser: Aliaser,
    _channels: ChannelRetriever,
    _labelClient: label.Client,
  ) {
    this.key = key;
    this.name = name;
    this.timeRange = timeRange;
    this.frameClient = _frameClient;
    this.color = color;
    this.kv = _kv;
    this.aliaser = _aliaser;
    this.channels = _channels;
    this.labelClient = _labelClient;
  }

  get payload(): Payload {
    return {
      key: this.key,
      name: this.name,
      timeRange: this.timeRange,
      color: this.color,
    };
  }

  async setAlias(channel: Key | Name, alias: string): Promise<void> {
    const ch = await this.channels.retrieve(channel);
    if (ch.length === 0) {
      throw new QueryError(`Channel ${channel} does not exist`);
    }
    await this.aliaser.set({ [ch[0].key]: alias });
  }

  async deleteAlias(...channels: Key[]): Promise<void> {
    await this.aliaser.delete(channels);
  }

  async listAliases(): Promise<Record<Key, string>> {
    return await this.aliaser.list();
  }

  async openAliasTracker(): Promise<signals.Observable<string, Alias>> {
    return await this.aliaser.openChangeTracker();
  }

  async read(channel: Key | Name): Promise<Series>;

  async read(channels: Params): Promise<framer.Frame>;

  async read(channels: Params): Promise<Series | framer.Frame> {
    return await this.frameClient.read(this.timeRange, channels);
  }

  async labels(): Promise<Label[]> {
    return await this.labelClient.retrieveFor(ontologyID(this.key));
  }

  async addLabel(...labels: label.Key[]): Promise<void> {
    await this.labelClient.label(ontologyID(this.key), labels);
  }

  async removeLabel(...labels: label.Key[]): Promise<void> {
    await this.labelClient.removeLabels(ontologyID(this.key), labels);
  }
}
