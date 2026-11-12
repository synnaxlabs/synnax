// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/list/List.css";

import { ranger } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { List as PList } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Access, type Flux } from "@synnaxlabs/pluto";
import { type ReactNode, useState } from "react";

import { ContextMenu } from "@/feature/range/list/ContextMenu";
import { Item, type ItemProps } from "@/feature/range/list/Item";
import { Empty } from "@/platform/empty";
import { Range } from "@/platform/range";

export interface ListProps
  extends
    Pick<
      Flux.UseListReturn<ranger.Key, ranger.Range>,
      "data" | "getItem" | "subscribe" | "more" | "answered"
    >,
    Pick<ItemProps, "showParent" | "showLabels" | "showTimeRange" | "showFavorite"> {
  emptyContent?: ReactNode;
}

const EmptyContent = () => {
  const openCreate = Range.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  return (
    <Empty.Action
      message="No ranges found"
      action={hasCreatePermission ? "Create range" : undefined}
      onClick={() => openCreate()}
    />
  );
};

const EMPTY_CONTENT = <EmptyContent />;

export const List = ({
  data,
  getItem,
  subscribe,
  more,
  answered,
  showParent = true,
  showLabels = true,
  showTimeRange = true,
  showFavorite = true,
  emptyContent = EMPTY_CONTENT,
}: ListProps) => {
  const [selected, setSelected] = useState<ranger.Key[]>([]);
  const menuProps = Menu.useContextMenu();
  return (
    <Flex.Box full="y" empty>
      <Select.Frame<ranger.Key, ranger.Range>
        multiple
        data={data}
        virtual
        getItem={getItem}
        subscribe={subscribe}
        onChange={setSelected}
        value={selected}
        onFetchMore={more}
        itemHeight={40}
      >
        <Menu.ContextMenu menu={contextMenu} {...menuProps} />
        <PList.Scroll grow onContextMenu={menuProps.open}>
          <PList.Items<string> emptyContent={answered && emptyContent}>
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
    </Flex.Box>
  );
};

const contextMenu = Component.renderProp<Menu.ContextMenuMenuProps>((p) => (
  <ContextMenu {...p} />
));
