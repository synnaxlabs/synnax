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
import { Dialog } from "@/dialog";
import { type Flux } from "@/flux";
import { type ListParams, useList } from "@/hardware/device/queries";
import { List } from "@/list";
import { Select } from "@/select";
import { Text } from "@/text";

const listItemRenderProp = Component.renderProp(
  ({ itemKey, ...rest }: List.ItemRenderProps<device.Key>) => {
    const item = List.useItem<device.Key, device.Device>(itemKey);
    return (
      <List.Item itemKey={itemKey} {...rest}>
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
  extends Select.SingleProps<device.Key, device.Device | undefined>,
    Flux.UseListArgs<ListParams, device.Key, device.Device> {}

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
        useListItem={useListItem}
        data={data}
        onChange={onChange}
      >
        <Select.SingleTrigger />
        <Select.Dialog<device.Key, ListParams>
          onSearch={retrieve}
          searchPlaceholder="Search Devices..."
          emptyContent={emptyContent}
        >
          {listItemRenderProp}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
