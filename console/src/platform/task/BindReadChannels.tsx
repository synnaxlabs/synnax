// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { type ReactElement } from "react";

import { BindChannels } from "@/platform/task/BindChannels";
import { type ReadChannelField, type ReadChannelProps } from "@/platform/task/channels";

export interface BindReadChannelsProps {
  /** The path of a read source, such as an endpoint, with `fields` and `index`. */
  path: string;
  /** The field of the source that keys it in the read properties of the device. */
  propertiesKeyField: string;
  /** The read properties of the device, or null while the device is unknown. */
  read: Record<string, ReadChannelProps> | null;
}

/**
 * Binds the fields of a read source to the channels the device stores for it: the index
 * field to the index, each other field to the channel of its pointer. Renders nothing.
 */
export const BindReadChannels = ({
  path,
  propertiesKeyField,
  read,
}: BindReadChannelsProps): ReactElement => {
  const propertiesKey = Form.useFieldValue<string>(`${path}.${propertiesKeyField}`);
  const indexKey = Form.useFieldValue<string>(`${path}.index`);
  const resolve = (field: ReadChannelField) => {
    if (read == null) return null;
    const props = read[propertiesKey];
    if (props == null) return { channel: 0 };
    return {
      channel:
        field.key === indexKey ? props.index : (props.channels[field.pointer] ?? 0),
    };
  };
  return <BindChannels<ReadChannelField> path={`${path}.fields`} resolve={resolve} />;
};
