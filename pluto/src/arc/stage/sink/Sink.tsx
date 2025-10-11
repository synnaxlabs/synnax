// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import z from "zod/v4";

import { Base } from "@/arc/stage/Base";
import { type types } from "@/arc/stage/types";
import { Channel } from "@/channel";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";

export const config = z.object({
  channel: channel.keyZ,
  value: z.number(),
});

export type Config = z.infer<typeof config>;

export interface SymbolProps extends types.SymbolProps<Config> {}

const FUCHSIA = color.construct("#FF00FF");
const FUCHSIA_BG = color.setAlpha(FUCHSIA, 0.2);
const FUCHSIA_TEXT = color.setAlpha(FUCHSIA, 1);

export const Symbol = ({ channel, value }: SymbolProps) => {
  const name =
    Channel.useRetrieve({ key: channel }, { addStatusOnFailure: false }).data?.name ??
    "Channel";

  return (
    <Base
      type="Sink"
      Icon={<Icon.Channel />}
      color={color.cssString(FUCHSIA_BG)}
      textColor={color.cssString(FUCHSIA_TEXT)}
      sinks={[{ key: "input", Icon: Icon.Value }]}
    >
      <Flex.Box x>
        <Text.Text
          level="p"
          weight={500}
          color={10}
          style={{ maxWidth: 100 }}
          overflow="ellipsis"
        >
          {name}
        </Text.Text>
        <Text.Text
          level="p"
          weight={500}
          color={10}
          style={{ maxWidth: 100 }}
          overflow="ellipsis"
          variant="code"
        >
          {value}
        </Text.Text>
      </Flex.Box>
    </Base>
  );
};
