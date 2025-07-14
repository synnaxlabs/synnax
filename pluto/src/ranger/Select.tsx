// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { type ReactElement, useCallback } from "react";

import { Component } from "@/component";
import { Dialog } from "@/dialog";
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
  extends Select.MultipleProps<ranger.Key, ranger.Payload | undefined> {}

export const SelectMultiple = ({
  onChange,
  value,
  emptyContent,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe } = useList();
  console.log(data);
  return (
    <Dialog.Frame {...rest} variant="connected">
      <Select.Frame<ranger.Key, ranger.Payload | undefined>
        multiple
        value={value}
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onChange={onChange}
        onFetchMore={useCallback(
          () =>
            retrieve(
              (p) => ({
                ...p,
                offset: (p?.offset ?? -10) + 10,
                limit: 10,
              }),
              { mode: "append" },
            ),
          [retrieve],
        )}
      >
        <Select.MultipleTrigger haulType={HAUL_TYPE} />
        <Select.Dialog<ranger.Key>
          onSearch={(term) => retrieve({ term, offset: 0, limit: 10 })}
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
  const { data, retrieve, subscribe, getItem } = useList({ filter, initialParams });
  const { onFetchMore, onSearch } = Flux.usePager({ retrieve });
  return (
    <Dialog.Frame variant="floating" {...rest}>
      <Select.Frame
        value={value}
        onChange={onChange}
        data={data}
        allowNone={allowNone}
        onFetchMore={onFetchMore}
        getItem={getItem}
        subscribe={subscribe}
      >
        <Select.SingleTrigger
          haulType={HAUL_TYPE}
          placeholder="Select a Range..."
          icon={<Icon.Range />}
        />
        <Select.Dialog<ranger.Key>
          onSearch={onSearch}
          searchPlaceholder="Search Ranges..."
          emptyContent={emptyContent}
          style={{ width: 500, height: 500 }}
        >
          {listItemRenderProp}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
