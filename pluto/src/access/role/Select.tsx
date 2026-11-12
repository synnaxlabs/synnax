// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access, type Synnax } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { type Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select as Base } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { icon } from "@/access/role/icon";
import { Flux } from "@/flux";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<access.role.Key>): ReactElement | null => {
  const item = List.useItem<access.role.Key, access.role.Role>(itemKey);
  if (item == null) return null;
  const { name, description } = item;
  return (
    <Base.Item itemKey={itemKey} {...rest}>
      {icon(item)}
      {name}
      {description !== "" && <Base.ItemDescription>{description}</Base.ItemDescription>}
    </Base.Item>
  );
};

// The trigger passes the selected role, or undefined while none is selected.
const renderIcon = (role: unknown): Icon.ReactElement | undefined =>
  role == null ? undefined : icon(role as access.role.Role);

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

export const Select = ({
  query,
  filter,
  triggerProps,
  ...props
}: SelectProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client: Synnax) => client.access.roles,
    { name: "roles", query, filter },
  );
  return (
    <Base.Single<access.role.Key, access.role.Role | undefined>
      {...props}
      triggerProps={{ renderIcon, ...triggerProps }}
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
