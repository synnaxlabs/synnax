// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/graph/node/sink/Sink.css";

import { color } from "@synnaxlabs/x";

import { Base } from "@/arc/graph/node/Base";
import { type Config } from "@/arc/graph/node/sink/config";
import { type types } from "@/arc/graph/node/types";
import { Channel } from "@/channel";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";

export interface SymbolProps extends types.SymbolProps<Config> {}

const FUCHSIA = color.construct("#FF00FF");
const FUCHSIA_BG = color.setAlpha(FUCHSIA, 0.2);
const FUCHSIA_TEXT = color.setAlpha(FUCHSIA, 1);

export const Symbol = ({ config: { channel, value } }: SymbolProps) => {
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
          className={CSS.BE("arc", "sink-label")}
          overflow="ellipsis"
        >
          {name}
        </Text.Text>
        <Text.Text
          level="p"
          weight={500}
          color={10}
          className={CSS.BE("arc", "sink-label")}
          overflow="ellipsis"
          variant="code"
        >
          {value}
        </Text.Text>
      </Flex.Box>
    </Base>
  );
};
