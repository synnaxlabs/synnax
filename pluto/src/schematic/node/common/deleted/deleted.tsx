// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel } from "@synnaxlabs/client";
import { Badge } from "@synnaxlabs/lyra/badge";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { Channel } from "@/channel";
import { Grid } from "@/schematic/node/common/grid";

/**
 * @returns true when the Core deleted the state or the command channel of an actuator.
 */
export const useEither = (state: channel.Key, command: channel.Key): boolean => {
  const stateDeleted = Channel.useDeleted(state);
  const commandDeleted = Channel.useDeleted(command);
  return stateDeleted || commandDeleted;
};

interface WatchProps {
  channel: channel.Key;
  onChange: (channel: channel.Key, deleted: boolean) => void;
}

const Watch = ({ channel, onChange }: WatchProps): null => {
  const deleted = Channel.useDeleted(channel);
  useEffect(() => onChange(channel, deleted), [channel, deleted, onChange]);
  return null;
};

const NONE: ReadonlySet<channel.Key> = new Set();

export interface Any {
  /** deleted is true when the Core deleted one of the channels. */
  deleted: boolean;
  /** watchers read whether each channel is deleted. The caller renders them. */
  watchers: ReactElement[];
}

/**
 * @returns whether the Core deleted any of channels, which can change in number, and
 * the elements that read it. A zero key is never deleted.
 */
export const useAny = (channels: channel.Key[]): Any => {
  const [deleted, setDeleted] = useState(NONE);
  const onChange = (key: channel.Key, gone: boolean): void =>
    setDeleted((prev) => {
      if (prev.has(key) === gone) return prev;
      const next = new Set(prev);
      if (gone) next.add(key);
      else next.delete(key);
      return next;
    });
  const keys = [...new Set(channels.filter(primitive.isNonZero))];
  return {
    deleted: keys.some((k) => deleted.has(k)),
    watchers: keys.map((k) => <Watch key={k} channel={k} onChange={onChange} />),
  };
};

/** Marks a symbol whose channel the Core deleted, below the symbol. */
export const Item = Grid.createItem((): ReactElement => (
  <Grid.Item itemKey="deleted" location="bottom" fixed>
    <Badge.Badge>{Channel.DELETED_TEXT}</Badge.Badge>
  </Grid.Item>
));
