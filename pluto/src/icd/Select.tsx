// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { type Flux } from "@/flux";
import { type ListQuery, useList } from "@/icd/queries";

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<icd.Key, icd.ICD | undefined>,
      "data" | "getItem" | "subscribe" | "children" | "resourceName"
    >,
    Flux.UseListParams<ListQuery, icd.Key, icd.ICD> {}

const ListItem = (props: List.ItemRenderProps<icd.Key>) => {
  const item = List.useItem<icd.Key, icd.ICD>(props.itemKey);
  return <Select.Item {...props}>{item?.name}</Select.Item>;
};

const listItemRenderProp = Component.renderProp(ListItem);

/** Selects one ICD. */
export const SelectSingle = ({
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
    <Select.Single<icd.Key, icd.ICD | undefined>
      resourceName="ICD"
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={fetchMore}
      onSearch={search}
      status={status}
      icon={<Icon.Library />}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
