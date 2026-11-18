// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration, type Synnax } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { Flux } from "@/flux";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<enumeration.Key>): ReactElement | null => {
  const item = List.useItem<enumeration.Key, enumeration.Enumeration>(itemKey);
  if (item == null) return null;
  return (
    <Select.Item itemKey={itemKey} {...rest}>
      <Icon.Enumeration />
      {item.name}
    </Select.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<enumeration.Key, enumeration.Enumeration | undefined>,
      "data" | "resourceName" | "subscribe" | "children"
    >,
    Omit<
      Flux.UseListParams<enumeration.RetrieveMultipleParams, enumeration.Enumeration>,
      "name"
    > {}

export const SelectSingle = ({
  filter,
  query,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client: Synnax) => client.enumerations,
    {
      name: "enumerations",
      query: query === null ? null : { order: { field: "name" }, ...query },
      filter,
    },
  );
  return (
    <Select.Single<enumeration.Key, enumeration.Enumeration | undefined>
      getItem={getItem}
      onFetchMore={more}
      onSearch={search}
      status={status}
      icon={<Icon.Enumeration />}
      {...rest}
      data={data}
      resourceName="enumeration"
      subscribe={subscribe}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
