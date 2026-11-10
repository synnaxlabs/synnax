// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/access/role/Select.css";

import { type access, type Synnax } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { List } from "@synnaxlabs/lyra/list";
import { Select as Base } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flux } from "@/flux";

const ListItem = ({
  itemKey,
  className,
  ...rest
}: List.ItemRenderProps<access.role.Key>): ReactElement | null => {
  const item = List.useItem<access.role.Key, access.role.Role>(itemKey);
  if (item == null) return null;
  const { name, description } = item;
  return (
    <Base.Item
      itemKey={itemKey}
      y
      gap="small"
      {...rest}
      className={CSS.cls(className, CSS.BE("access-role", "item"))}
    >
      <Text.Text level="p">{name}</Text.Text>
      {item?.description != null && (
        <Text.Text level="small" color={9} overflow="wrap">
          {description}
        </Text.Text>
      )}
    </Base.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectProps
  extends
    Omit<
      Base.SingleProps<access.role.Key, access.role.Role | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Omit<
      Flux.UseListParams<access.role.RetrieveMultipleParams, access.role.Role>,
      "name"
    > {}

export const Select = ({ query, filter, ...props }: SelectProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client: Synnax) => client.access.roles,
    { name: "roles", query, filter },
  );
  return (
    <Base.Single<access.role.Key, access.role.Role | undefined>
      {...props}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={more}
      onSearch={search}
      status={status}
      resourceName="role"
      virtual={false}
    >
      {listItemRenderProp}
    </Base.Single>
  );
};
