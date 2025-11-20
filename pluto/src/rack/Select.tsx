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
import { Icon } from "@/icon";
import { List } from "@/list";
import { type ListQuery, useList } from "@/rack/queries";
import { StatusIndicator } from "@/rack/StatusIndicator";
import { Select } from "@/select";
import { Text } from "@/text";

export interface SelectSingleProps
  extends Omit<Select.SingleFrameProps<rack.Key, rack.Payload | undefined>, "data">,
    Flux.UseListParams<ListQuery, rack.Key, rack.Payload>,
    Omit<Dialog.FrameProps, "onChange">,
    Pick<Select.DialogProps<rack.Key>, "emptyContent"> {}

const listItemRenderProp = Component.renderProp(
  (props: List.ItemRenderProps<rack.Key>) => {
    const { itemKey } = props;
    const item = List.useItem<rack.Key, rack.Rack>(itemKey);
    return (
      <Select.ListItem {...props} align="center" justify="between">
        <Text.Text>{item?.name}</Text.Text>
        <StatusIndicator status={item?.status} tooltipLocation="left" />
      </Select.ListItem>
    );
  },
);

export const SelectSingle = ({
  filter,
  initialQuery,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    initialQuery: { includeStatus: true, ...initialQuery },
    filter,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Single<rack.Key, rack.Payload | undefined>
      resourceName="Driver"
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={fetchMore}
      onSearch={search}
      status={status}
      icon={<Icon.Rack />}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
