// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/charon/component";
import { Icon } from "@synnaxlabs/charon/icon";
import { List } from "@synnaxlabs/charon/list";
import { Select as Base } from "@synnaxlabs/charon/select";
import { Status } from "@synnaxlabs/charon/status";
import { Text } from "@synnaxlabs/charon/text";
import { type status } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { type Flux } from "@/flux";
import { type ListParams, useList } from "@/status/queries";
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
    Flux.UseListParams<ListParams, status.Key, status.Status> {}

export const Select = ({
  initialQuery,
  filter,
  ...props
}: SelectProps): ReactElement => {
  const { data, retrieve, subscribe, getItem, status } = useList({
    initialQuery,
    filter,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Base.Single<status.Key, status.Status>
      {...props}
      resourceName="status"
      data={data}
      subscribe={subscribe}
      getItem={getItem}
      icon={<Icon.Status />}
      onFetchMore={fetchMore}
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
    <Base.ListItem {...props}>
      <Text.Text level="p">
        <Status.Indicator variant={variant} />
        {name}
      </Text.Text>
    </Base.ListItem>
  );
};

export const listItemRenderProp = Component.renderProp(ListItem);
