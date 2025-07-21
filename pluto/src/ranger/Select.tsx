// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Component } from "@/component";
import { Flux } from "@/flux";
import { Icon } from "@/icon";
import { List } from "@/list";
import { type ListParams, useList } from "@/ranger/queries";
import { TimeRangeChip } from "@/ranger/TimeRangeChip";
import { HAUL_TYPE } from "@/ranger/types";
import { Select } from "@/select";
import { Text } from "@/text";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<ranger.Key>): ReactElement | null => {
  const item = List.useItem<ranger.Key, ranger.Payload>(itemKey);
  const { selected, onSelect, hovered } = Select.useItemState<ranger.Key>(itemKey);
  if (item == null) return null;
  return (
    <List.Item
      itemKey={itemKey}
      onSelect={onSelect}
      selected={selected}
      hovered={hovered}
      justify="spaceBetween"
      {...rest}
    >
      <Text.Text
        level="p"
        shade={10}
        weight={450}
        style={{
          maxWidth: 250,
          textOverflow: "ellipsis",
          overflow: "hidden",
          whiteSpace: "nowrap",
        }}
      >
        {item?.name}
      </Text.Text>
      <TimeRangeChip level="small" timeRange={item.timeRange} />
    </List.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectMultipleProps
  extends Omit<
      Select.MultipleProps<ranger.Key, ranger.Payload | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Flux.UseListArgs<ListParams, ranger.Key, ranger.Payload> {}

export const SelectMultiple = ({
  onChange,
  value,
  emptyContent,
  filter,
  initialParams,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, ...status } = useList({
    filter,
    initialParams,
  });
  const { onFetchMore, onSearch } = Flux.usePager({ retrieve });
  return (
    <Select.Multiple<ranger.Key, ranger.Payload | undefined>
      resourceName="Range"
      haulType={HAUL_TYPE}
      value={value}
      onChange={onChange}
      data={data}
      getItem={getItem}
      icon={<Icon.Range />}
      subscribe={subscribe}
      onFetchMore={onFetchMore}
      onSearch={onSearch}
      emptyContent={emptyContent}
      status={status}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Multiple>
  );
};

export interface SelectSingleProps
  extends Omit<
      Select.SingleProps<ranger.Key, ranger.Payload | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Flux.UseListArgs<ListParams, ranger.Key, ranger.Payload> {}

export const SelectSingle = ({
  onChange,
  value,
  filter,
  allowNone,
  emptyContent,
  initialParams,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, subscribe, getItem, ...status } = useList({
    filter,
    initialParams,
  });
  const { onFetchMore, onSearch } = Flux.usePager({ retrieve });
  return (
    <Select.Single<ranger.Key, ranger.Payload | undefined>
      resourceName="Range"
      variant="modal"
      value={value}
      onChange={onChange}
      data={data}
      allowNone={allowNone}
      haulType={HAUL_TYPE}
      onFetchMore={onFetchMore}
      getItem={getItem}
      subscribe={subscribe}
      status={status}
      onSearch={onSearch}
      emptyContent={emptyContent}
      icon={<Icon.Range />}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};

const DATA: Select.SimplyEntry<ranger.Stage>[] = [
  { key: "to_do", name: "To Do", icon: <Icon.ToDo /> },
  { key: "in_progress", name: "In Progress", icon: <Icon.InProgress /> },
  { key: "completed", name: "Completed", icon: <Icon.Completed /> },
];

export interface SelectStageProps extends Select.SimpleProps<ranger.Stage> {}

export const SelectStage = (props: SelectStageProps): ReactElement => (
  <Select.Simple {...props} data={DATA} resourceName="Stage" />
);
