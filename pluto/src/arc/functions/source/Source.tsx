// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import z from "zod/v4";

import { Base } from "@/arc/functions/Base";
import { type types } from "@/arc/functions/types";
import { Channel } from "@/channel";
import { Icon } from "@/icon";
import { Text } from "@/text";

export const config = z.object({
  channel: channel.keyZ,
});

export type Config = z.infer<typeof config>;

export interface SymbolProps extends types.SymbolProps<Config> {}

export const Symbol = ({ channel }: SymbolProps) => {
  const name =
    Channel.useRetrieve({ key: channel }, { addStatusOnFailure: false }).data?.name ??
    "Channel";

  return (
    <Base
      type="Source"
      Icon={<Icon.Channel />}
      color="var(--pluto-primary-z-20)"
      textColor="var(--pluto-primary-z)"
      sources={[{ key: "output", Icon: Icon.Value }]}
    >
      <Text.Text
        level="p"
        weight={500}
        color={10}
        style={{ maxWidth: 100 }}
        overflow="ellipsis"
      >
        {name}
      </Text.Text>
    </Base>
  );
};
