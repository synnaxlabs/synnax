// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/device/Select.css";

import { type device } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { CSS } from "@synnaxlabs/lyra/css";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type ListParams, useList } from "@/device/queries";
import { StatusIndicator } from "@/device/StatusIndicator";
import { type Flux } from "@/flux";
const listItemRenderProp = Component.renderProp(
  ({ itemKey, ...rest }: List.ItemRenderProps<device.Key>) => {
    const item = List.useItem<device.Key, device.Device>(itemKey);
    if (item == null) return null;
    const { name, location, status } = item;
    return (
      <Select.ListItem
        itemKey={itemKey}
        {...rest}
        className={CSS.BE("device", "list-item")}
        justify="between"
        align="center"
      >
        <Text.Text align="center">
          <StatusIndicator status={status} />
          {name}
        </Text.Text>
        <Text.Text level="small" color={9} weight={450} style={LOCATION_STYLE}>
          {location}
        </Text.Text>
      </Select.ListItem>
    );
  },
);

const LOCATION_STYLE = { marginTop: "0.25rem" } as const;

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<device.Key, device.Device | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Flux.UseListParams<ListParams, device.Key, device.Device> {}

export const SelectSingle = ({
  filter,
  initialQuery,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    filter,
    initialQuery,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Single<device.Key, device.Device | undefined>
      resourceName="device"
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={fetchMore}
      onSearch={search}
      status={status}
      icon={<Icon.Device />}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
