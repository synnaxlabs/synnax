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
import { useCallback, useEffect } from "react";

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
  const { data, retrieve } = Channel.useRetrieveStateful();
  useEffect(() => {
    if (primitive.isZero(channel)) return;
    retrieve({ key: channel, rangeKey: range ?? undefined });
  }, [channel, range]);
  const { update } = Channel.useRename();
  const name = data?.name ?? defaultName;
  const handleRename = useCallback(
    (name: string) => update({ key: channel, name }),
    [channel, update],
  );
  return (
    <Text.MaybeEditable
      className={CSS(className, CSS.BE("task", "channel-name"))}
      status={primitive.isZero(channel) ? "warning" : undefined}
      level="small"
      value={name}
      onChange={handleRename}
      allowDoubleClick={false}
      overflow="ellipsis"
      {...rest}
    />
  );
};
