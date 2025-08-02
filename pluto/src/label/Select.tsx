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
import { type ListParams, useList } from "@/label/queries";
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
      <Text.Text level="p" align="center">
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
      "data" | "multiple" | "resourceName" | "subscribe" | "children"
    >,
    Flux.UseListArgs<ListParams, label.Key, label.Label> {}

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

export const SelectMultiple = ({
  onChange,
  value,
  emptyContent,
  filter,
  initialParams,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    filter,
    initialParams,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Multiple<label.Key, label.Label | undefined>
      resourceName="Label"
      haulType={HAUL_TYPE}
      value={value}
      onChange={onChange}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={fetchMore}
      onSearch={search}
      emptyContent={emptyContent}
      status={status}
      renderTag={labelRenderTag}
      icon={<Icon.Label />}
      triggerVariant="text"
      variant="floating"
      {...rest}
    >
      {listItemRenderProp}
    </Select.Multiple>
  );
};

export interface SelectSingleProps
  extends Omit<
      Select.SingleProps<label.Key, label.Label | undefined>,
      "data" | "useListItem" | "resourceName" | "subscribe" | "children"
    >,
    Flux.UseListArgs<ListParams, label.Key, label.Label> {}

export const SelectSingle = ({
  onChange,
  value,
  allowNone,
  emptyContent,
  filter,
  initialParams,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    filter,
    initialParams,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Single<label.Key, label.Label | undefined>
      resourceName="Label"
      value={value}
      onChange={onChange}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      allowNone={allowNone}
      onFetchMore={fetchMore}
      onSearch={search}
      emptyContent={emptyContent}
      status={status}
      haulType={HAUL_TYPE}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
