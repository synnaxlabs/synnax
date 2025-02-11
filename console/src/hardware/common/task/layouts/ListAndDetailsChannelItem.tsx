// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, List, Text } from "@synnaxlabs/pluto";
import { type Key, type Keyed } from "@synnaxlabs/x";

import { ChannelName } from "@/hardware/common/task/ChannelName";
import { EnableDisableButton } from "@/hardware/common/task/EnableDisableButton";
import { TareButton } from "@/hardware/common/task/TareButton";

export interface ListAndDetailsChannelItemProps<K extends Key, E extends Keyed<K>>
  extends List.ItemProps<K, E> {
  port: string | number;
  portMaxChars: number;
  canTare: boolean;
  channel: number;
  onTare?: (channel: number) => void;
  isSnapshot: boolean;
  path: string;
  hasTareButton: boolean;
  name?: string;
}

const NAME_PROPS: Text.TextProps = {
  level: "p",
  shade: 7,
  weight: 450,
  style: {
    maxWidth: 150,
    flexGrow: 1,
    textOverflow: "ellipsis",
    overflow: "hidden",
  },
  noWrap: true,
};

export const ListAndDetailsChannelItem = <K extends Key, E extends Keyed<K>>({
  port,
  portMaxChars,
  canTare,
  onTare,
  isSnapshot,
  path,
  hasTareButton,
  channel,
  name,
  ...rest
}: ListAndDetailsChannelItemProps<K, E>) => (
  <List.ItemFrame
    {...rest}
    justify="spaceBetween"
    align="center"
    style={{ padding: "1.25rem 2rem" }}
  >
    <Align.Space direction="x" size="small">
      <Text.Text
        level="p"
        shade={6}
        weight={500}
        style={{ width: `${portMaxChars * 1.25}rem` }}
      >
        {port}
      </Text.Text>
      {name != null ? (
        <Text.Text {...NAME_PROPS}>{name}</Text.Text>
      ) : (
        <ChannelName {...NAME_PROPS} channel={channel} />
      )}
    </Align.Space>
    <Align.Pack direction="x" align="center" size="small">
      {hasTareButton && (
        <TareButton disabled={!canTare} onTare={() => onTare?.(channel)} />
      )}
      <EnableDisableButton path={`${path}.enabled`} isSnapshot={isSnapshot} />
    </Align.Pack>
  </List.ItemFrame>
);
