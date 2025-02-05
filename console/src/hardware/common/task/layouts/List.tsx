// Copyright 2025 Synnax Labs, Inc.
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
  ChannelList,
  type ChannelListProps,
} from "@/hardware/common/task/layouts/ChannelList";

export interface ListProps<C extends Channel>
  extends Pick<ChannelListProps<C>, "generateChannel" | "isSnapshot" | "children"> {}

export const List = <C extends Channel>({
  children,
  isSnapshot,
  generateChannel,
}: ListProps<C>): ReactElement => {
  const [selected, setSelected] = useState<string[]>([]);
  return (
    <ChannelList<C>
      isSnapshot={isSnapshot}
      selected={selected}
      onSelect={setSelected}
      generateChannel={generateChannel}
    >
      {children}
    </ChannelList>
  );
};
