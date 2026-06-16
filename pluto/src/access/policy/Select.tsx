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

import { type ListParams, useList } from "@/access/policy/queries";
import { HAUL_TYPE } from "@/access/policy/types";
import { Component } from "@/component";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { List } from "@/list";
import { Select } from "@/select";
import { Text } from "@/text";

const listItemRenderProp = Component.renderProp(
  ({
    itemKey,
    ...rest
  }: List.ItemRenderProps<access.policy.Key>): ReactElement | null => {
    const item = List.useItem<access.policy.Key, access.policy.Policy>(itemKey);
    if (item == null) return null;
    const { name } = item;
    return (
      <Select.ListItem itemKey={itemKey} y gap="small" {...rest}>
        <Text.Text level="p" weight={400}>
          {name}
        </Text.Text>
      </Select.ListItem>
    );
  },
);

export interface SelectMultipleProps
  extends
    Omit<
      Select.MultipleProps<access.policy.Key, access.policy.Policy | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Flux.UseListParams<ListParams, access.policy.Key, access.policy.Policy> {}

export const SelectMultiple = ({
  initialQuery,
  filter,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    initialQuery,
    filter,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Multiple<access.policy.Key, access.policy.Policy | undefined>
      haulType={HAUL_TYPE}
      onFetchMore={fetchMore}
      onSearch={search}
      status={status}
      icon={<Icon.Policy />}
      {...rest}
      resourceName="policy"
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      itemHeight={56}
    >
      {listItemRenderProp}
    </Select.Multiple>
  );
};

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<access.policy.Key, access.policy.Policy | undefined>,
      "data" | "getItem" | "subscribe" | "children" | "resourceName"
    >,
    Flux.UseListParams<ListParams, access.policy.Key, access.policy.Policy> {}

export const SelectSingle = ({
  initialQuery,
  filter,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    initialQuery,
    filter,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Single<access.policy.Key, access.policy.Policy | undefined>
      onFetchMore={fetchMore}
      onSearch={search}
      status={status}
      haulType={HAUL_TYPE}
      icon={<Icon.Policy />}
      {...rest}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      resourceName="policy"
      itemHeight={56}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
