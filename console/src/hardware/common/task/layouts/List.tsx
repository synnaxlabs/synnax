// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { type Channel } from "@/hardware/common/task/ChannelList";
import {
  DefaultChannelList,
  type DefaultChannelListProps,
} from "@/hardware/common/task/DefaultChannelList";

export interface ListProps<C extends Channel>
  extends Pick<
    DefaultChannelListProps<C>,
    "generateChannel" | "isSnapshot" | "children"
  > {}

export const List = <C extends Channel>({
  children,
  isSnapshot,
  generateChannel,
}: ListProps<C>): ReactElement => {
  const [selected, setSelected] = useState<string[]>([]);
  return (
    <DefaultChannelList<C>
      isSnapshot={isSnapshot}
      selected={selected}
      onSelect={setSelected}
      generateChannel={generateChannel}
    >
      {(p) => children(p)}
    </DefaultChannelList>
  );
};
