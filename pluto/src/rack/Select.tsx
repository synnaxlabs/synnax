// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { type Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { type Flux } from "@/flux";
import { type ListQuery, useList } from "@/rack/queries";
import { StatusIndicator } from "@/rack/StatusIndicator";

export interface SelectSingleProps
  extends
    Omit<Select.SingleFrameProps<rack.Key, rack.Payload | undefined>, "data">,
    Omit<Flux.UseListParams<ListQuery, rack.Rack>, "name">,
    Omit<Dialog.FrameProps, "onChange">,
    Pick<Select.ItemsProps<rack.Key>, "emptyContent">,
    Pick<
      Select.SingleProps<rack.Key, rack.Payload | undefined>,
      "preview" | "triggerProps" | "dialogProps"
    > {}

const ListItem = (props: List.ItemRenderProps<rack.Key>) => {
  const { itemKey } = props;
  const item = List.useItem<rack.Key, rack.Rack>(itemKey);
  return (
    <Select.Item {...props} align="center">
      <Text.Text el="span" overflow="fade" grow>
        {item?.name}
      </Text.Text>
      <StatusIndicator status={item?.status} tooltipLocation="left" />
    </Select.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export const SelectSingle = ({
  filter,
  query,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = useList({
    query,
    filter,
  });
  return (
    <Select.Single<rack.Key, rack.Payload | undefined>
      resourceName="driver"
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={more}
      onSearch={search}
      status={status}
      icon={<Icon.Rack />}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
