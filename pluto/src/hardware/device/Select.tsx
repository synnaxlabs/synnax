// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Breadcrumb } from "@/breadcrumb";
import { Component } from "@/component";
import { Flux } from "@/flux";
import { type ListParams, useList } from "@/hardware/device/queries";
import { List } from "@/list";
import { Select } from "@/select";
import { Text } from "@/text";

const listItemRenderProp = Component.renderProp(
  ({ itemKey, ...rest }: List.ItemRenderProps<device.Key>) => {
    const item = List.useItem<device.Key, device.Device>(itemKey);
    const selectProps = Select.useItemState(itemKey);
    return (
      <List.Item itemKey={itemKey} {...rest} {...selectProps}>
        <Text.Text level="p">{item?.name}</Text.Text>
        <Breadcrumb.Breadcrumb
          level="small"
          shade={9}
          weight={450}
          style={{ marginTop: "0.25rem" }}
          size="tiny"
        >
          {item?.location ?? ""}
        </Breadcrumb.Breadcrumb>
      </List.Item>
    );
  },
);

export interface SelectSingleProps
  extends Omit<
      Select.SingleProps<device.Key, device.Device | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Flux.UseListArgs<ListParams, device.Key, device.Device> {}

export const SelectSingle = ({
  onChange,
  value,
  filter,
  allowNone,
  emptyContent,
  initialParams,
  disabled,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, ...status } = useList({
    filter,
    initialParams,
  });
  const { onFetchMore, onSearch } = Flux.usePager({ retrieve });
  return (
    <Select.Single<device.Key, device.Device | undefined>
      resourceName="Device"
      value={value}
      onChange={onChange}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={onFetchMore}
      onSearch={onSearch}
      emptyContent={emptyContent}
      status={status}
      disabled={disabled}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
