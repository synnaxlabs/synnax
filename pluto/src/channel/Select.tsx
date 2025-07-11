// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { DataType } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { useAliases } from "@/channel/AliasContext";
import { type ListParams, useList } from "@/channel/queries";
import { HAUL_TYPE } from "@/channel/types";
import { Component } from "@/component";
import { Dialog } from "@/dialog";
import { Icon } from "@/icon";
import { List } from "@/list";
import { Select } from "@/select";
import { Text } from "@/text";

export const resolveIcon = (ch?: channel.Payload): Icon.FC => {
  if (ch == null) return Icon.Channel;
  if (channel.isCalculated(ch)) return Icon.Calculation;
  if (ch.isIndex) return Icon.Index;
  const dt = new DataType(ch.dataType);
  if (dt.isInteger) return Icon.Binary;
  if (dt.isFloat) return Icon.Decimal;
  if (dt.equals(DataType.STRING)) return Icon.String;
  if (dt.equals(DataType.JSON)) return Icon.JSON;
  return Icon.Channel;
};

const listItemRenderProp = Component.renderProp(
  ({ itemKey, ...rest }: List.ItemRenderProps<channel.Key>): ReactElement | null => {
    const item = List.useItem<channel.Key, channel.Channel>(itemKey);
    const { selected, onSelect, hovered } = Select.useItemState<channel.Key>(itemKey);
    const aliases = useAliases();
    const Icon = resolveIcon(item?.payload);
    const displayName = aliases[item?.key ?? 0] ?? item?.name ?? "";
    return (
      <List.Item
        itemKey={itemKey}
        onSelect={onSelect}
        selected={selected}
        hovered={hovered}
        {...rest}
      >
        <Align.Space direction="x" size="small" align="center">
          <Icon />
          <Text.Text level="p">{displayName}</Text.Text>
        </Align.Space>
      </List.Item>
    );
  },
);

export interface SelectMultipleProps
  extends Omit<
      Select.MultipleFrameProps<channel.Key, channel.Channel | undefined>,
      "data" | "useListItem" | "multiple"
    >,
    Pick<Select.DialogProps<channel.Key, ListParams>, "emptyContent">,
    Omit<Dialog.FrameProps, "onChange"> {
  searchOptions?: channel.RetrieveOptions;
}

export const SelectMultiple = ({
  onChange,
  value,
  emptyContent,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, useListItem, retrieve } = useList();
  return (
    <Select.Frame<channel.Key, channel.Channel | undefined>
      multiple
      value={value}
      onChange={onChange}
      useListItem={useListItem}
      data={data}
      {...rest}
    >
      <Select.MultipleTrigger haulType={HAUL_TYPE} />
      <Select.Dialog<channel.Key, ListParams>
        onSearch={retrieve}
        searchPlaceholder="Search channels..."
        emptyContent={emptyContent}
      >
        {listItemRenderProp}
      </Select.Dialog>
    </Select.Frame>
  );
};

export interface SelectSingleProps
  extends Omit<
      Select.SingleFrameProps<channel.Key, channel.Channel | undefined>,
      "data" | "useListItem"
    >,
    Pick<Select.DialogProps<channel.Key, ListParams>, "emptyContent">,
    Omit<Dialog.FrameProps, "onChange"> {
  searchOptions?: channel.RetrieveOptions;
}

export const SelectSingle = ({
  onChange,
  value,
  allowNone,
  emptyContent,
  className,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, useListItem, retrieve } = useList();
  return (
    <Dialog.Frame {...rest}>
      <Select.Frame<channel.Key, channel.Channel | undefined>
        value={value}
        onChange={onChange}
        data={data}
        useListItem={useListItem}
        allowNone={allowNone}
      >
        <Select.SingleTrigger haulType={HAUL_TYPE} icon={<Icon.Channel />} />
        <Select.Dialog<channel.Key, ListParams>
          onSearch={retrieve}
          searchPlaceholder="Search channels..."
          emptyContent={emptyContent}
        >
          {listItemRenderProp}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
