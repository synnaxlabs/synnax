// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/views/List.css";

import { type Component } from "@synnaxlabs/lyra/component";
import { type List as PList } from "@synnaxlabs/lyra/list";
import { useState } from "react";

import { CSS } from "@/platform/css";
import { HoverProvider } from "@/platform/task/port/Hovered";
import { type Channel } from "@/platform/task/types";
import {
  ChannelList,
  type ChannelListProps,
  Header,
} from "@/platform/task/views/ChannelList";
import { LIST_CHANNEL_COLUMNS } from "@/platform/task/views/ListChannelItem";
import { type PortsProps, type View, ViewSwitch } from "@/platform/task/views/Ports";

export interface ListProps<C extends Channel>
  extends
    Pick<
      ChannelListProps<C>,
      | "listItem"
      | "contextMenuItems"
      | "path"
      | "resolve"
      | "header"
      | "checkable"
      | "emptyContent"
      | "addLabel"
      | "tareable"
    >,
    Required<Pick<ChannelListProps<C>, "createChannel">> {
  /**
   * The task's own columns, for items that are `ListChannelItem`s. The arrow and
   * channel columns follow them.
   */
  columns?: PList.Column[];
  /** The channel columns after the task's own. */
  channelColumns?: PList.Column[];
  /**
   * The channels by port. The header becomes a "Channels" title with a "List" and
   * "Ports" switch that shows it.
   */
  ports?: Component.RenderProp<PortsProps<C>>;
  /**
   * The channels by port, shown between the title and the items. The pointer on an item
   * marks its port, and the other way around.
   */
  map?: Component.RenderProp<PortsProps<C>>;
}

export const List = <C extends Channel>({
  columns,
  channelColumns = LIST_CHANNEL_COLUMNS,
  ports,
  map,
  header,
  ...rest
}: ListProps<C>) => {
  const [selected, setSelected] = useState<string[]>([]);
  const [view, setView] = useState<View>("list");
  const list = (
    <ChannelList<C>
      {...rest}
      header={
        map != null ? (
          <>
            <Header />
            <div className={CSS.BE("task-list", "map")}>
              {map({ path: rest.path, selected, onSelect: setSelected })}
            </div>
          </>
        ) : ports == null ? (
          header
        ) : (
          <Header actions={<ViewSwitch value={view} onChange={setView} />} />
        )
      }
      columns={columns && [...columns, ...channelColumns]}
      ports={
        ports != null && view === "ports"
          ? ports({ path: rest.path, selected, onSelect: setSelected })
          : undefined
      }
      selected={selected}
      onSelect={setSelected}
    />
  );
  return map == null ? list : <HoverProvider>{list}</HoverProvider>;
};
