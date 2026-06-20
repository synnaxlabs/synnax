// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Base } from "@/arc/graph/node/Base";
import { type Config } from "@/arc/graph/node/source/config";
import { type types } from "@/arc/graph/node/types";
import { Channel } from "@/channel";
import { Icon } from "@/icon";
import { Text } from "@/text";

export interface SymbolProps extends types.SymbolProps<Config> {}

export const Symbol = ({ config: { channel } }: SymbolProps) => {
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
