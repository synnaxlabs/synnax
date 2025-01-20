// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type change } from "@synnaxlabs/x/change";
import { type observe } from "@synnaxlabs/x/observe";
import { type Series } from "@synnaxlabs/x/telem";

import { type channel } from "@/channel";
import { framer } from "@/framer";

export interface Decoder<K, V> {
  (variant: change.Variant, data: Series): Array<change.Change<K, V>>;
}

export interface Observable<K, V>
  extends observe.ObservableAsyncCloseable<Array<change.Change<K, V>>> {}

export const openObservable = async <K, V>(
  client: framer.Client,
  setChannel: channel.Key | channel.Name,
  deleteChannel: channel.Key | channel.Name,
  codec: Decoder<K, V>,
): Promise<Observable<K, V>> => {
  const stream = await client.openStreamer([setChannel, deleteChannel] as channel.Keys);
  const transform = (frame: framer.Frame): [Array<change.Change<K, V>>, boolean] => {
    const changes: Array<change.Change<K, V>> = [];
    if (deleteChannel != null)
      changes.push(
        ...frame.get(deleteChannel).series.flatMap((s) => codec("delete", s)),
      );
    if (setChannel != null)
      changes.push(...frame.get(setChannel).series.flatMap((s) => codec("set", s)));
    return [changes, changes.length > 0];
  };
  return new framer.ObservableStreamer<Array<change.Change<K, V>>>(stream, transform);
};
