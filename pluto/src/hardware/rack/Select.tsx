// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Component } from "@/component";
import { type Dialog } from "@/dialog";
import { type Flux } from "@/flux";
import { Rack } from "@/hardware/rack";
import { useList } from "@/hardware/rack/queries";
import { Icon } from "@/icon";
import { List } from "@/list";
import { type ListParams } from "@/ranger/queries";
import { Select } from "@/select";
import { Text } from "@/text";

export interface SelectSingleProps
  extends Omit<
      Select.SingleFrameProps<rack.Key, rack.Payload | undefined>,
      "data" | "useListItem"
    >,
    Flux.UseListArgs<ListParams, rack.Key, rack.Payload>,
    Omit<Dialog.FrameProps, "onChange">,
    Pick<Select.DialogProps<rack.Key>, "emptyContent"> {}

const listItemRenderProp = Component.renderProp(
  (props: List.ItemRenderProps<rack.Key>) => {
    const { itemKey } = props;
    const item = List.useItem<rack.Key, rack.Rack>(itemKey);
    return (
      <Select.ListItem {...props} align="center" justify="spaceBetween">
        <Text.Text level="p">{item?.name}</Text.Text>
        <Rack.StatusIndicator status={item?.status} tooltipLocation="left" />
      </Select.ListItem>
    );
  },
);

export const SelectSingle = ({
  value,
  onChange,
  filter,
  allowNone,
  emptyContent,
  initialParams,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, ...status } = useList({
    initialParams: { includeStatus: true, ...initialParams },
    filter,
  });
  const { onFetchMore, onSearch } = List.usePager({ retrieve });
  return (
    <Select.Single<rack.Key, rack.Payload | undefined>
      resourceName="Driver"
      value={value}
      onChange={onChange}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={onFetchMore}
      onSearch={onSearch}
      emptyContent={emptyContent}
      status={status}
      icon={<Icon.Rack />}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
