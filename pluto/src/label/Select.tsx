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
import { Dialog } from "@/dialog";
import { Flux } from "@/flux";
import { Icon } from "@/icon";
import { useList } from "@/label/queries";
import { HAUL_TYPE } from "@/label/types";
import { List } from "@/list";
import { Select } from "@/select";
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
      <Icon.Circle color={item?.color} size="1.5rem" />
      <Text.Text level="p">{item?.name}</Text.Text>
    </List.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectMultipleProps
  extends Omit<
      Select.MultipleFrameProps<label.Key, label.Label | undefined>,
      "data" | "useListItem" | "multiple"
    >,
    Pick<Select.DialogProps<label.Key>, "emptyContent">,
    Omit<Dialog.FrameProps, "onChange"> {}

export const SelectMultiple = ({
  onChange,
  value,
  emptyContent,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe } = useList();
  const { onFetchMore, onSearch } = Flux.usePager({ retrieve });
  return (
    <Dialog.Frame {...rest}>
      <Select.Frame<label.Key, label.Label | undefined>
        multiple
        value={value}
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onChange={onChange}
        onFetchMore={onFetchMore}
      >
        <Select.MultipleTrigger haulType={HAUL_TYPE} />
        <Select.Dialog<label.Key>
          onSearch={onSearch}
          searchPlaceholder="Search labels..."
          emptyContent={emptyContent}
        >
          {listItemRenderProp}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};

export interface SelectSingleProps
  extends Omit<
      Select.SingleFrameProps<label.Key, label.Label | undefined>,
      "data" | "useListItem"
    >,
    Omit<Dialog.FrameProps, "onChange">,
    Pick<Select.DialogProps<label.Key>, "emptyContent"> {}

export const SelectSingle = ({
  onChange,
  value,
  allowNone,
  emptyContent,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe } = useList();
  const { onFetchMore, onSearch } = Flux.usePager({ retrieve });
  return (
    <Dialog.Frame {...rest}>
      <Select.Frame
        value={value}
        onChange={onChange}
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        allowNone={allowNone}
        onFetchMore={onFetchMore}
      >
        <Select.SingleTrigger haulType={HAUL_TYPE} />
        <Select.Dialog<label.Key>
          onSearch={onSearch}
          searchPlaceholder="Search labels..."
          emptyContent={emptyContent}
        >
          {listItemRenderProp}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
