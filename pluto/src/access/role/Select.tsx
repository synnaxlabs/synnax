// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { type ListQuery, useList } from "@/access/role/queries";
import { Component } from "@/component";
import { type Flux } from "@/flux";
import { List } from "@/list";
import { Select as Core } from "@/select";
import { Text } from "@/text";

const listItemRenderProp = Component.renderProp(
  ({
    itemKey,
    ...rest
  }: List.ItemRenderProps<access.role.Key>): ReactElement | null => {
    const item = List.useItem<access.role.Key, access.role.Role>(itemKey);
    if (item == null) return null;
    const { name, description } = item;
    return (
      <Core.ListItem
        itemKey={itemKey}
        y
        gap="small"
        {...rest}
        style={{ overflow: "hidden", maxWidth: "100%" }}
      >
        <Text.Text level="p" weight={400}>
          {name}
        </Text.Text>
        {item?.description != null && (
          <Text.Text level="small" color={9} overflow="wrap">
            {description}
          </Text.Text>
        )}
      </Core.ListItem>
    );
  },
);

export interface SelectProps
  extends
    Omit<
      Core.SingleProps<access.role.Key, access.role.Role | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Flux.UseListParams<ListQuery, access.role.Key, access.role.Role> {}

export const Select = ({
  initialQuery,
  filter,
  ...props
}: SelectProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    initialQuery,
    filter,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Core.Single<access.role.Key, access.role.Role | undefined>
      {...props}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={fetchMore}
      onSearch={search}
      status={status}
      resourceName="role"
      virtual={false}
    >
      {listItemRenderProp}
    </Core.Single>
  );
};
