// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Component } from "@/component";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { type ListQuery, useList } from "@/label/queries";
import { HAUL_TYPE } from "@/label/types";
import { List } from "@/list";
import { Select } from "@/select";
import { Tag } from "@/tag";
import { Text } from "@/text";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<label.Key>): ReactElement | null => {
  const item = List.useItem<label.Key, label.Label>(itemKey);
  const { selected, onSelect, hovered } = Select.useItemState<label.Key>(itemKey);
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
        <Icon.Circle color={item?.color} size="2.5em" />
        {item?.name}
      </Text.Text>
    </List.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectMultipleProps
  extends Omit<
      Select.MultipleProps<label.Key, label.Label | undefined>,
      "data" | "resourceName" | "subscribe" | "children"
    >,
    Flux.UseListParams<ListQuery, label.Key, label.Label> {}

const labelRenderTag = Component.renderProp(
  (props: Select.MultipleTagProps<label.Key>): ReactElement | null => {
    const { itemKey } = props;
    const item = List.useItem<label.Key, label.Label>(itemKey);
    const { onSelect } = Select.useItemState<label.Key>(itemKey);
    if (item == null) return null;
    return (
      <Tag.Tag color={item.color} onClose={onSelect} size="small">
        {item.name}
      </Tag.Tag>
    );
  },
);

const SELECT_MULTIPLE_TRIGGER_PROPS: Select.MultipleTriggerProps<label.Key> = {
  variant: "text",
};

export const SelectMultiple = ({
  filter,
  initialQuery,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    filter,
    initialQuery,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Multiple<label.Key, label.Label | undefined>
      haulType={HAUL_TYPE}
      getItem={getItem}
      onFetchMore={fetchMore}
      onSearch={search}
      status={status}
      renderTag={labelRenderTag}
      icon={<Icon.Label />}
      triggerProps={SELECT_MULTIPLE_TRIGGER_PROPS}
      variant="floating"
      {...rest}
      data={data}
      subscribe={subscribe}
      resourceName="label"
    >
      {listItemRenderProp}
    </Select.Multiple>
  );
};

export interface SelectSingleProps
  extends Omit<
      Select.SingleProps<label.Key, label.Label | undefined>,
      "data" | "resourceName" | "subscribe" | "children"
    >,
    Flux.UseListParams<ListQuery, label.Key, label.Label> {}

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
    <Select.Single<label.Key, label.Label | undefined>
      getItem={getItem}
      onFetchMore={fetchMore}
      onSearch={search}
      status={status}
      haulType={HAUL_TYPE}
      {...rest}
      data={data}
      resourceName="label"
      subscribe={subscribe}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
