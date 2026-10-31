// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Component } from "@synnaxlabs/lyra/component";
import { useState } from "react";

import { type Channel } from "@/platform/task/types";
import {
  ChannelList,
  type ChannelListProps,
  Header,
} from "@/platform/task/views/ChannelList";
import { type PortsProps, type View, ViewSwitch } from "@/platform/task/views/Ports";

export interface ListProps<C extends Channel> extends Pick<
  ChannelListProps<C>,
  "createChannel" | "listItem" | "contextMenuItems" | "resolve"
> {
  /** The channels by port. A "List" and "Ports" switch in the header shows it. */
  ports?: Component.RenderProp<PortsProps>;
}

export const List = <C extends Channel>({ ports, ...rest }: ListProps<C>) => {
  const [selected, setSelected] = useState<string[]>([]);
  const [view, setView] = useState<View>("list");
  return (
    <ChannelList<C>
      {...rest}
      header={
        ports == null ? undefined : (
          <Header actions={<ViewSwitch value={view} onChange={setView} />} />
        )
      }
      ports={
        ports != null && view === "ports"
          ? ports({ selected, onSelect: setSelected })
          : undefined
      }
      selected={selected}
      onSelect={setSelected}
    />
  );
};
