// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type Flux } from "@/flux";
import { type ListQuery, useList } from "@/schematic/symbol/queries";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<schematic.symbol.Key>): ReactElement | null => {
  const item = List.useItem<schematic.symbol.Key, schematic.symbol.Symbol>(itemKey);
  if (item == null) return null;
  return (
    <Select.ListItem itemKey={itemKey} {...rest}>
      <Text.Text align="center">
        <Icon.Schematic />
        {item.name}
      </Text.Text>
    </Select.ListItem>
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
  const { data, retrieve, getItem, subscribe, status } = useList({
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
