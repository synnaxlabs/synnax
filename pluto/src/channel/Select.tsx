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

import { type ListQuery, useList } from "@/channel/queries";
import { HAUL_TYPE } from "@/channel/types";
import { Component } from "@/component";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { List } from "@/list";
import { Select } from "@/select";

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
    const Icon = resolveIcon(item?.payload);
    return (
      <Select.ListItem itemKey={itemKey} {...rest}>
        <Icon />
        {item?.name}
      </Select.ListItem>
    );
  },
);

export interface SelectMultipleProps
  extends Omit<
      Select.MultipleProps<channel.Key, channel.Channel | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Flux.UseListParams<ListQuery, channel.Key, channel.Channel> {}

export const SelectMultiple = ({
  onChange,
  value,
  emptyContent,
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
    <Select.Multiple<channel.Key, channel.Channel | undefined>
      resourceName="Channel"
      value={value}
      onChange={onChange}
      data={data}
      haulType={HAUL_TYPE}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={fetchMore}
      onSearch={search}
      emptyContent={emptyContent}
      status={status}
      icon={<Icon.Channel />}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Multiple>
  );
};

export interface SelectSingleProps
  extends Omit<
      Select.SingleProps<channel.Key, channel.Channel | undefined>,
      "data" | "getItem" | "subscribe" | "children" | "resourceName"
    >,
    Flux.UseListParams<ListQuery, channel.Key, channel.Channel> {}

export const SelectSingle = ({
  onChange,
  value,
  allowNone,
  emptyContent,
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
    <Select.Single<channel.Key, channel.Channel | undefined>
      resourceName="Channel"
      onChange={onChange}
      value={value}
      allowNone={allowNone}
      emptyContent={emptyContent}
      onFetchMore={fetchMore}
      onSearch={search}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      status={status}
      haulType={HAUL_TYPE}
      icon={<Icon.Channel />}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
