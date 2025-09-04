// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Channel, Text } from "@synnaxlabs/pluto";
import { type Optional, primitive } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { useSelectActiveKey as useSelectActiveRangeKey } from "@/range/selectors";

export interface ChannelNameProps
  extends Optional<Omit<Text.MaybeEditableProps, "value">, "level"> {
  channel: channel.Key;
  defaultName?: string;
}

export const ChannelName = ({
  channel,
  defaultName = "No Channel",
  className,
  ...rest
}: ChannelNameProps) => {
  const range = useSelectActiveRangeKey();
  const { data } = Channel.retrieve.useDirect({
    params: { key: channel, rangeKey: range ?? undefined },
  });
  const { update: rename } = Channel.rename.useDirect({ params: { key: channel } });
  const name = data?.name ?? defaultName;
  return (
    <Text.MaybeEditable
      className={CSS(className, CSS.BE("task", "channel-name"))}
      status={primitive.isZero(channel) ? "warning" : undefined}
      level="small"
      value={name}
      onChange={rename}
      allowDoubleClick={false}
      {...rest}
    />
  );
};
