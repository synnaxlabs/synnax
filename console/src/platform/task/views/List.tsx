// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Component } from "@synnaxlabs/lyra/component";
import { type List as PList } from "@synnaxlabs/lyra/list";
import { useState } from "react";

import { type Channel } from "@/platform/task/types";
import {
  ChannelList,
  type ChannelListProps,
  Header,
} from "@/platform/task/views/ChannelList";
import { LIST_CHANNEL_COLUMNS } from "@/platform/task/views/ListChannelItem";
import { type PortsProps, type View, ViewSwitch } from "@/platform/task/views/Ports";

export interface ListProps<C extends Channel> extends Pick<
  ChannelListProps<C>,
  | "createChannel"
  | "listItem"
  | "contextMenuItems"
  | "path"
  | "resolve"
  | "header"
  | "checkable"
> {
  /**
   * The task's own columns, for items that are `ListChannelItem`s. The arrow and
   * channel columns follow them.
   */
  columns?: PList.Column[];
  /**
   * The channels by port. The header becomes a "Channels" title with a "List" and
   * "Ports" switch that shows it.
   */
  ports?: Component.RenderProp<PortsProps<C>>;
}

export const List = <C extends Channel>({
  columns,
  ports,
  header,
  ...rest
}: ListProps<C>) => {
  const [selected, setSelected] = useState<string[]>([]);
  const [view, setView] = useState<View>("list");
  return (
    <ChannelList<C>
      {...rest}
      header={
        ports == null ? (
          header
        ) : (
          <Header actions={<ViewSwitch value={view} onChange={setView} />} />
        )
      }
      columns={columns && [...columns, ...LIST_CHANNEL_COLUMNS]}
      ports={
        ports != null && view === "ports"
          ? ports({ path: rest.path, selected, onSelect: setSelected })
          : undefined
      }
      selected={selected}
      onSelect={setSelected}
    />
  );
};
