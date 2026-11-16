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
    <Select.Item itemKey={itemKey} {...rest}>
      <Icon.Schematic />
      {item.name}
    </Select.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<schematic.symbol.Key, schematic.symbol.Symbol | undefined>,
      "data" | "resourceName" | "subscribe" | "children"
    >,
    Omit<Flux.UseListParams<ListQuery, schematic.symbol.Symbol>, "name"> {}

export const SelectSingle = ({
  filter,
  query,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = useList({
    filter,
    query,
  });
  return (
    <Select.Single<schematic.symbol.Key, schematic.symbol.Symbol | undefined>
      getItem={getItem}
      onFetchMore={more}
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
