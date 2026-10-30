// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/list/List.css";

import { type ranger } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { List as PList } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { type Flux } from "@synnaxlabs/pluto";
import { useState } from "react";

import { ContextMenu } from "@/feature/range/list/ContextMenu";
import { Item, type ItemProps } from "@/feature/range/list/Item";

export type ListProps = Pick<
  Flux.UseListReturn<PList.PagerParams, ranger.Key, ranger.Range>,
  "data" | "getItem" | "subscribe" | "retrieve"
> &
  Pick<ItemProps, "showParent" | "showLabels" | "showTimeRange" | "showFavorite">;

export const List = ({
  data,
  getItem,
  subscribe,
  retrieve,
  showParent = true,
  showLabels = true,
  showTimeRange = true,
  showFavorite = true,
}: ListProps) => {
  const [selected, setSelected] = useState<ranger.Key[]>([]);
  const menuProps = Menu.useContextMenu();
  const { fetchMore } = PList.usePager({ retrieve, pageSize: 25 });
  return (
    <Select.Frame<ranger.Key, ranger.Range>
      multiple
      data={data}
      virtual
      getItem={getItem}
      subscribe={subscribe}
      onChange={setSelected}
      value={selected}
      onFetchMore={fetchMore}
      itemHeight={40}
    >
      <Menu.ContextMenu menu={contextMenu} {...menuProps} />
      <PList.Scroll grow onContextMenu={menuProps.open}>
        <PList.Items<string>>
          {({ key, ...rest }) => (
            <Item
              key={key}
              {...rest}
              showParent={showParent}
              showLabels={showLabels}
              showTimeRange={showTimeRange}
              showFavorite={showFavorite}
            />
          )}
        </PList.Items>
      </PList.Scroll>
    </Select.Frame>
  );
};

const contextMenu = Component.renderProp<Menu.ContextMenuMenuProps>((p) => (
  <ContextMenu {...p} />
));
