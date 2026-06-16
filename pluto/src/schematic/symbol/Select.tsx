// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Component } from "@/component";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { List } from "@/list";
import { type ListQuery, useList } from "@/schematic/symbol/queries";
import { Select } from "@/select";
import { Text } from "@/text";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<schematic.symbol.Key>): ReactElement | null => {
  const item = List.useItem<schematic.symbol.Key, schematic.symbol.Symbol>(itemKey);
  const { selected, onSelect, hovered } =
    Select.useItemState<schematic.symbol.Key>(itemKey);
  if (item == null) return null;
  return (
    <List.Item
      itemKey={itemKey}
      onSelect={onSelect}
      selected={selected}
      hovered={hovered}
      {...rest}
    >
      <Text.Text align="center">
        <Icon.Schematic />
        {item.name}
      </Text.Text>
    </List.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<schematic.symbol.Key, schematic.symbol.Symbol | undefined>,
      "data" | "resourceName" | "subscribe" | "children"
    >,
    Flux.UseListParams<ListQuery, schematic.symbol.Key, schematic.symbol.Symbol> {}

export const SelectSingle = ({
  filter,
  initialQuery,
  ...rest
}: SelectSingleProps): ReactElement => {
  const {
    data,
    retrieve,
    getItem,
    subscribe,
    stat: status,
  } = useList({
    filter,
    initialQuery,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Single<schematic.symbol.Key, schematic.symbol.Symbol | undefined>
      getItem={getItem}
      onFetchMore={fetchMore}
      onSearch={search}
      status={status}
      {...rest}
      data={data}
      resourceName="symbol"
      subscribe={subscribe}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
