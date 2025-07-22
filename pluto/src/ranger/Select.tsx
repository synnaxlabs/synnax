// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { Breadcrumb } from "@/breadcrumb";
import { Component } from "@/component";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { List } from "@/list";
import { type ListParams, useList } from "@/ranger/queries";
import { TimeRangeChip } from "@/ranger/TimeRangeChip";
import { HAUL_TYPE } from "@/ranger/types";
import { Select } from "@/select";
import { Tag } from "@/tag";

const ListItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<ranger.Key>): ReactElement | null => {
  const item = List.useItem<ranger.Key, ranger.Payload>(itemKey);
  if (item == null) return null;
  const { name, timeRange, parent, labels } = item;
  const breadcrumbSegments: Breadcrumb.Segments = [
    {
      label: name,
      weight: 450,
      shade: 10,
    },
  ];
  if (parent != null)
    breadcrumbSegments.push({
      label: parent.name,
      weight: 400,
      shade: 8,
    });
  return (
    <Select.ListItem itemKey={itemKey} justify="spaceBetween" {...rest}>
      <Align.Space y size="small">
        <Breadcrumb.Breadcrumb>{breadcrumbSegments}</Breadcrumb.Breadcrumb>
        <TimeRangeChip level="small" timeRange={timeRange} />
      </Align.Space>
      <Tag.Tags>
        {labels?.map((l) => (
          <Tag.Tag key={l.key} color={l.color} size="small">
            {l.name}
          </Tag.Tag>
        ))}
      </Tag.Tags>
    </Select.ListItem>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectMultipleProps
  extends Omit<
      Select.MultipleProps<ranger.Key, ranger.Payload | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Flux.UseListArgs<ListParams, ranger.Key, ranger.Payload> {}

export const SelectMultiple = ({
  onChange,
  value,
  emptyContent,
  filter,
  initialParams,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, ...status } = useList({
    filter,
    initialParams,
  });
  const { onFetchMore, onSearch } = List.usePager({ retrieve });
  return (
    <Select.Multiple<ranger.Key, ranger.Payload | undefined>
      resourceName="Range"
      haulType={HAUL_TYPE}
      value={value}
      onChange={onChange}
      data={data}
      getItem={getItem}
      icon={<Icon.Range />}
      subscribe={subscribe}
      onFetchMore={onFetchMore}
      onSearch={onSearch}
      emptyContent={emptyContent}
      status={status}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Multiple>
  );
};

export interface SelectSingleProps
  extends Omit<
      Select.SingleProps<ranger.Key, ranger.Payload | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Flux.UseListArgs<ListParams, ranger.Key, ranger.Payload> {}

export const SelectSingle = ({
  onChange,
  value,
  filter,
  allowNone,
  emptyContent,
  initialParams,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, subscribe, getItem, ...status } = useList({
    filter,
    initialParams,
  });
  const { onFetchMore, onSearch } = List.usePager({ retrieve });
  return (
    <Select.Single<ranger.Key, ranger.Payload | undefined>
      resourceName="Range"
      variant="modal"
      value={value}
      onChange={onChange}
      data={data}
      allowNone={allowNone}
      haulType={HAUL_TYPE}
      onFetchMore={onFetchMore}
      getItem={getItem}
      subscribe={subscribe}
      status={status}
      onSearch={onSearch}
      emptyContent={emptyContent}
      icon={<Icon.Range />}
      itemHeight={56}
      dialogProps={{
        style: {
          width: 800,
        },
      }}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
