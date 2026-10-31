// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type List as PList } from "@synnaxlabs/lyra/list";
import { useState } from "react";

import { type Channel } from "@/platform/task/types";
import { ChannelList, type ChannelListProps } from "@/platform/task/views/ChannelList";
import { LIST_CHANNEL_COLUMNS } from "@/platform/task/views/ListChannelItem";

export interface ListProps<C extends Channel> extends Pick<
  ChannelListProps<C>,
  "createChannel" | "listItem" | "contextMenuItems" | "resolve" | "header" | "checkable"
> {
  /**
   * The task's own columns, for items that are `ListChannelItem`s. The arrow and
   * channel columns follow them.
   */
  columns?: PList.Column[];
}

export const List = <C extends Channel>({ columns, ...rest }: ListProps<C>) => {
  const [selected, setSelected] = useState<string[]>([]);
  return (
    <ChannelList<C>
      {...rest}
      columns={columns && [...columns, ...LIST_CHANNEL_COLUMNS]}
      selected={selected}
      onSelect={setSelected}
    />
  );
};
