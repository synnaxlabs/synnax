// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type group, type Synnax } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { Flux } from "@/flux";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<group.Key>): ReactElement | null => {
  const item = List.useItem<group.Key, group.Group>(itemKey);
  if (item == null) return null;
  return (
    <Select.Item itemKey={itemKey} {...rest}>
      <Icon.Group />
      {item.name}
    </Select.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<group.Key, group.Group | undefined>,
      "data" | "resourceName" | "subscribe" | "children"
    >,
    Omit<Flux.UseListParams<group.RetrieveMultipleParams, group.Group>, "name"> {}

export const SelectSingle = ({
  filter,
  query,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client: Synnax) => client.groups,
    { name: "groups", query, filter },
  );
  return (
    <Select.Single<group.Key, group.Group | undefined>
      getItem={getItem}
      onFetchMore={more}
      onSearch={search}
      status={status}
      {...rest}
      data={data}
      resourceName="group"
      subscribe={subscribe}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
