// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status, type Synnax } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select as Base } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { Flux } from "@/flux";

export interface SelectProps
  extends
    Omit<
      Base.SingleProps<status.Key, status.Status>,
      | "data"
      | "getItem"
      | "subscribe"
      | "status"
      | "onFetchMore"
      | "onSearch"
      | "children"
      | "resourceName"
    >,
    Omit<Flux.UseListParams<status.RetrieveMultipleParams, status.Status>, "name"> {}

export const Select = ({ query, filter, ...props }: SelectProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client: Synnax) => client.statuses,
    { name: "statuses", query, filter },
  );
  return (
    <Base.Single<status.Key, status.Status>
      {...props}
      resourceName="status"
      data={data}
      subscribe={subscribe}
      getItem={getItem}
      icon={<Icon.Status />}
      onFetchMore={more}
      onSearch={search}
      status={status}
    >
      {listItemRenderProp}
    </Base.Single>
  );
};

const ListItem = (props: List.ItemProps<status.Key>): ReactElement | null => {
  const { itemKey } = props;
  const item = List.useItem<status.Key, status.Status>(itemKey);
  if (item == null) return null;
  const { name, variant } = item;
  return (
    <Base.Item {...props}>
      <Text.Text level="p">
        <Status.Indicator variant={variant} />
        {name}
      </Text.Text>
    </Base.Item>
  );
};

export const listItemRenderProp = Component.renderProp(ListItem);
