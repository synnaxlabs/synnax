// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { observe, type Destructor, type change, type Series } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { type framer } from "@/framer";

export type Decoder<K, V> = (
  variant: change.Variant,
  data: Series,
) => Array<change.Change<K, V>>;

export class Observable<K, V>
  implements observe.Observable<Array<change.Change<K, V>>>
{
  private readonly streamer: framer.Streamer;
  private readonly decoder: Decoder<K, V>;
  private readonly base: observe.Observer<Array<change.Change<K, V>>>;
  private readonly closePromise: Promise<void>;
  private readonly deleteChannel?: channel.Key | channel.Name;
  private readonly setChannel?: channel.Key | channel.Name;

  private constructor(
    streamer: framer.Streamer,
    ecd: Decoder<K, V>,
    setChannel?: channel.Key | channel.Name,
    deleteChannel?: channel.Key | channel.Name,
  ) {
    this.streamer = streamer;
    this.decoder = ecd;
    this.base = new observe.Observer<Array<change.Change<K, V>>>();
    this.closePromise = this.stream();
    this.deleteChannel = deleteChannel;
    this.setChannel = setChannel;
  }

  onChange(handler: observe.Handler<Array<change.Change<K, V>>>): Destructor {
    return this.base.onChange(handler);
  }

  async close(): Promise<void> {
    this.streamer.close();
    return await this.closePromise;
  }

  async stream(): Promise<void> {
    for await (const frame of this.streamer) {
      const changes: Array<change.Change<K, V>> = [];
      if (this.deleteChannel != null) {
        const deletes = frame.get(this.deleteChannel);
        changes.push(...deletes.flatMap((s) => this.decoder("delete", s)));
      }
      if (this.setChannel != null) {
        const sets = frame.get(this.setChannel);
        changes.push(...sets.flatMap((s) => this.decoder("set", s)));
      }
      this.base.notify(changes);
    }
  }

  static async open<K, V>(
    client: framer.Client,
    setChannel: channel.Key | channel.Name,
    deleteChannel: channel.Key | channel.Name,
    ecd: Decoder<K, V>,
  ): Promise<Observable<K, V>> {
    const stream = await client.newStreamer([
      setChannel,
      deleteChannel,
    ] as channel.Keys);
    return new Observable(stream, ecd, setChannel, deleteChannel);
  }
}
