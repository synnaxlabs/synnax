// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, query, type Synnax as Client } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { DataType } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { resolveIcon } from "@/channel/resolveIcon";
import { Telem } from "@/telem";

export interface HaulPreviewProps {
  channel: channel.Payload;
  /** The channel's index, which the preview names when it is given. */
  index?: channel.Payload;
  /** How many items the drag carries. */
  count?: number;
  /** Whether a calculation writes the channel, or undefined when it is not known. */
  calculated?: boolean;
}

const resolveKind = (
  ch: channel.Payload,
  calculated: boolean | undefined,
  index?: channel.Payload,
): ReactNode => {
  if (ch.isIndex) return "Index";
  if (calculated === true) return "Calculated";
  if (ch.virtual) return "Virtual";
  if (index == null) return null;
  const IndexIcon = resolveIcon(index);
  return <Haul.Fact icon={<IndexIcon />}>{index.name}</Haul.Fact>;
};

/** The drag preview of a channel: its name, data type, and kind or index. */
export const HaulPreview = ({
  channel: ch,
  index,
  count,
  calculated,
}: HaulPreviewProps): ReactElement => {
  const Icon = resolveIcon(ch, calculated);
  const kind = resolveKind(ch, calculated, index);
  const facts: ReactNode[] = [Telem.resolveDataTypeName(new DataType(ch.dataType))];
  if (kind != null) facts.push(kind);
  return <Haul.Preview icon={<Icon />} name={ch.name} facts={facts} count={count} />;
};

/** Returns the index of a channel from the client cache. It never fetches. */
export const getCachedIndex = (
  client: Client | null,
  ch: channel.Payload,
): channel.Payload | undefined => {
  if (client == null || ch.index === 0 || ch.isIndex) return undefined;
  const index = client.channels.watch(ch.index).getCached();
  return query.isLive(index) ? index.payload : undefined;
};

/**
 * Returns whether a calculation writes the channel, from the client cache. It never
 * fetches, since a drag preview draws at once, so it returns undefined when the cache
 * holds no answer.
 */
export const getCachedIsOutput = (
  client: Client | null,
  ch: channel.Payload,
): boolean | undefined => {
  const output = client?.calculations.outputs.get(ch.key);
  return output == null ? undefined : output.calculation != null;
};
