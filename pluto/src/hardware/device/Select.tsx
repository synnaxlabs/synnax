// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/device/Select.css";

import { type device } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Breadcrumb } from "@/breadcrumb";
import { Component } from "@/component";
import { CSS } from "@/css";
import { type Flux } from "@/flux";
import { Device } from "@/hardware/device";
import { type ListParams, useList } from "@/hardware/device/queries";
import { Icon } from "@/icon";
import { List } from "@/list";
import { Select } from "@/select";
import { Text } from "@/text";

const listItemRenderProp = Component.renderProp(
  ({ itemKey, ...rest }: List.ItemRenderProps<device.Key>) => {
    const item = List.useItem<device.Key, device.Device>(itemKey);
    return (
      <Select.ListItem
        itemKey={itemKey}
        {...rest}
        className={CSS.BE("device", "list-item")}
        justify="between"
        align="center"
      >
        <Text.Text align="center">
          <Device.StatusIndicator status={item?.status} />
          {item?.name}
        </Text.Text>
        <Breadcrumb.Breadcrumb
          level="small"
          color={9}
          weight={450}
          style={{ marginTop: "0.25rem" }}
          gap="tiny"
        >
          {item?.location.split(".").map((segment) => (
            <Breadcrumb.Segment key={segment}>{segment}</Breadcrumb.Segment>
          ))}
        </Breadcrumb.Breadcrumb>
      </Select.ListItem>
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
  icon = <Icon.Device />,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    filter,
    initialParams,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Single<device.Key, device.Device | undefined>
      resourceName="Device"
      value={value}
      onChange={onChange}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={fetchMore}
      onSearch={search}
      emptyContent={emptyContent}
      status={status}
      disabled={disabled}
      icon={icon}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
