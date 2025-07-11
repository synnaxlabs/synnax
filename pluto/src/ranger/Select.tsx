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
import { Dialog } from "@/dialog";
import { type Flux } from "@/flux";
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
  extends Select.MultipleProps<ranger.Key, ranger.Payload | undefined> {}

export const SelectMultiple = ({
  onChange,
  value,
  emptyContent,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, retrieve, useListItem } = useList();
  return (
    <Dialog.Frame {...rest}>
      <Select.Frame<ranger.Key, ranger.Payload | undefined>
        multiple
        value={value}
        data={data}
        useListItem={useListItem}
        onChange={onChange}
      >
        <Select.MultipleTrigger haulType={HAUL_TYPE} />
        <Select.Dialog<ranger.Key, ListParams>
          onSearch={retrieve}
          searchPlaceholder="Search Ranges..."
          emptyContent={emptyContent}
        >
          {listItemRenderProp}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};

export interface SelectSingleProps
  extends Select.SingleProps<ranger.Key, ranger.Payload | undefined>,
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
  const { data, useListItem, retrieve } = useList({ filter, initialParams });
  return (
    <Dialog.Frame {...rest}>
      <Select.Frame
        value={value}
        onChange={onChange}
        data={data}
        useListItem={useListItem}
        allowNone={allowNone}
        // onFetchMore={() => retrieve({ ...initialParams, offset: data.length })}
      >
        <Select.SingleTrigger
          haulType={HAUL_TYPE}
          placeholder="Select a Range..."
          icon={<Icon.Range />}
        />
        <Select.Dialog<ranger.Key, ListParams>
          onSearch={retrieve}
          searchPlaceholder="Search Ranges..."
          emptyContent={emptyContent}
          style={{
            width: 500,
            height: 500,
          }}
        >
          {listItemRenderProp}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
