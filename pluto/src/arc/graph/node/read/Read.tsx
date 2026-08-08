// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/graph/node/read/Read.css";

import { Handle } from "@/arc/graph/handle";
import { type Config } from "@/arc/graph/node/read/config";
import { type types } from "@/arc/graph/node/types";
import { Channel } from "@/channel";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";

export interface SymbolProps extends types.SymbolProps<Config> {}

export const Symbol = ({ config: { channel } }: SymbolProps) => {
  const name = Channel.useRetrieve({ key: channel }).data?.name ?? "Channel";

  return (
    <Flex.Box
      pack
      x
      align="center"
      background={1}
      bordered
      borderColor={6}
      rounded="small"
    >
      <Flex.Box
        className={CSS.BE("arc", "read-icon-frame")}
        align="center"
        justify="center"
      >
        <Icon.Channel className={CSS.BE("arc", "read-icon")} />
      </Flex.Box>
      <Divider.Divider y color={5} />
      <Flex.Box className={CSS.BE("arc", "read-body")} align="start" empty>
        <Text.Text level="small" weight={500} color={9}>
          Read Channel
        </Text.Text>
        <Text.Text level="h4" weight={450} color={10}>
          {name}
        </Text.Text>
      </Flex.Box>
      <Handle.Sink location="left" id="trigger" />
      <Handle.Source location="right" id="output" />
    </Flex.Box>
  );
};
