// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type library } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { type Flux } from "@/flux";
import { type ListQuery, useList } from "@/library/queries";

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<library.Key, library.Library | undefined>,
      "data" | "getItem" | "subscribe" | "children" | "resourceName"
    >,
    Flux.UseListParams<ListQuery, library.Key, library.Library> {}

const listItemRenderProp = Component.renderProp(
  (props: List.ItemRenderProps<library.Key>) => {
    const item = List.useItem<library.Key, library.Library>(props.itemKey);
    return <Select.Item {...props}>{item?.name}</Select.Item>;
  },
);

/** Selects one library. */
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
    <Select.Single<library.Key, library.Library | undefined>
      resourceName="library"
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
