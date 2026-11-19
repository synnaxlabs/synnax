// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { BindChannels } from "@/platform/task/BindChannels";
import { type ReadChannelField, type ReadChannelProps } from "@/platform/task/channels";

type BoundField = Pick<ReadChannelField, "key" | "pointer" | "channel">;

/** A read source, such as an endpoint, whose fields read into channels. */
interface ReadSource {
  index: string;
  fields: BoundField[];
}

export interface BindReadChannelsProps {
  /** The path of a read source. */
  path: deep.Target<ReadSource>;
  /** The path of the source field that keys it in the read properties of the device. */
  propertiesKey: deep.Target<string>;
  /** The read properties of the device, or null while the device is unknown. */
  read: Record<string, ReadChannelProps> | null;
  /**
   * Changes each bound field to agree with the channel it binds. It must return the
   * field itself when nothing changes.
   */
  match?: (field: BoundField, channel: channel.Channel) => BoundField;
}

/**
 * Binds the fields of a read source to the channels the device stores for it: the index
 * field to the index, each other field to the channel of its pointer. Renders nothing.
 */
export const BindReadChannels = ({
  path,
  propertiesKey,
  read,
  match,
}: BindReadChannelsProps): ReactElement => {
  const source = deep.widen(path);
  const key = Form.useFieldValue(propertiesKey);
  const indexKey = Form.useFieldValue(source.index);
  const resolve = (field: BoundField): Partial<BoundField> | null => {
    if (read == null) return null;
    const props = read[key];
    if (props == null) return { channel: 0 };
    return {
      channel:
        field.key === indexKey ? props.index : (props.channels[field.pointer] ?? 0),
    };
  };
  const matchChannel =
    match == null
      ? undefined
      : (field: BoundField, channels: channel.Channel[]): BoundField => {
          const bound = channels.find(({ key }) => key === field.channel);
          return bound == null ? field : match(field, bound);
        };
  return <BindChannels path={source.fields} resolve={resolve} match={matchChannel} />;
};
