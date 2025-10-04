// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Component } from "@/component";
import { type Flux } from "@/flux";
import { List } from "@/list";
import { Select as Core } from "@/select";
import { type ListParams, useList } from "@/status/queries";
import { Text } from "@/text";

export interface SelectProps
  extends Omit<
      Core.SingleProps<status.Key, status.Status>,
      | "data"
      | "getItem"
      | "subscribe"
      | "status"
      | "onFetchMore"
      | "onSearch"
      | "children"
      | "resourceName"
    >,
    Flux.UseListParams<ListParams, status.Key, status.Status> {}

export const Select = ({
  value,
  onChange,
  filter,
  initialQuery,
  ...props
}: SelectProps): ReactElement => {
  const { data, retrieve, subscribe, getItem, status } = useList({ initialQuery });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Core.Single<status.Key, status.Status>
      resourceName="Status"
      data={data}
      subscribe={subscribe}
      getItem={getItem}
      onFetchMore={fetchMore}
      onSearch={search}
      value={value}
      onChange={onChange}
      status={status}
      {...props}
    >
      {listItemRenderProp}
    </Core.Single>
  );
};

const ListItem = (props: List.ItemProps<status.Key>): ReactElement | null => {
  const { itemKey } = props;
  const item = List.useItem<status.Key, status.Status>(itemKey);
  if (item == null) return null;
  const { name } = item;
  return (
    <Core.ListItem {...props}>
      <Text.Text level="p">{name}</Text.Text>
    </Core.ListItem>
  );
};

export const listItemRenderProp = Component.renderProp(ListItem);
