// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { memo, type ReactElement } from "react";

import { Align } from "@/align";
import { Breadcrumb } from "@/breadcrumb";
import { Component } from "@/component";
import { type Dialog } from "@/dialog";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { List } from "@/list";
import { type ListParams, useList } from "@/ranger/queries";
import { TimeRangeChip } from "@/ranger/TimeRangeChip";
import { HAUL_TYPE } from "@/ranger/types";
import { Select } from "@/select";
import { Tag } from "@/tag";

interface ListItemProps extends List.ItemProps<ranger.Key> {
  showParent?: boolean;
  showLabels?: boolean;
}

export const ListItem = memo(
  ({
    itemKey,
    showParent = true,
    showLabels = true,
    ...rest
  }: ListItemProps): ReactElement | null => {
    const item = List.useItem<ranger.Key, ranger.Payload>(itemKey);
    if (item == null) return null;
    const { name, parent, timeRange, labels } = item;
    const breadcrumbSegments: Breadcrumb.Segments = [
      {
        label: name,
        weight: 450,
        shade: 10,
      },
    ];
    if (parent != null && showParent)
      breadcrumbSegments.push({
        label: parent.name,
        weight: 400,
        shade: 8,
      });
    return (
      <Select.ListItem x itemKey={itemKey} justify="spaceBetween" {...rest}>
        <Breadcrumb.Breadcrumb>{breadcrumbSegments}</Breadcrumb.Breadcrumb>
        <Align.Space x>
          <TimeRangeChip level="small" timeRange={timeRange} />
          {showLabels && (
            <Tag.Tags>
              {labels?.map((l) => (
                <Tag.Tag key={l.key} color={l.color} size="small">
                  {l.name}
                </Tag.Tag>
              ))}
            </Tag.Tags>
          )}
        </Align.Space>
      </Select.ListItem>
    );
  },
);
ListItem.displayName = "Ranger.ListItem";

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectMultipleProps
  extends Omit<
      Select.MultipleProps<ranger.Key, ranger.Payload | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Flux.UseListArgs<ListParams, ranger.Key, ranger.Payload> {}

const ICON = <Icon.Range />;

export const SelectMultiple = ({
  onChange,
  value,
  emptyContent,
  filter,
  initialParams,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    filter,
    initialParams,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Multiple<ranger.Key, ranger.Payload | undefined>
      resourceName="Range"
      haulType={HAUL_TYPE}
      value={value}
      onChange={onChange}
      data={data}
      getItem={getItem}
      icon={ICON}
      subscribe={subscribe}
      onFetchMore={fetchMore}
      onSearch={search}
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

const DIALOG_PROPS: Dialog.DialogProps = {
  style: { width: 800 },
};

export const SelectSingle = ({
  onChange,
  value,
  filter,
  allowNone,
  emptyContent,
  initialParams,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, subscribe, getItem, status } = useList({
    filter,
    initialParams,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Single<ranger.Key, ranger.Payload | undefined>
      resourceName="Range"
      variant="modal"
      value={value}
      onChange={onChange}
      data={data}
      allowNone={allowNone}
      haulType={HAUL_TYPE}
      onFetchMore={fetchMore}
      getItem={getItem}
      subscribe={subscribe}
      status={status}
      onSearch={search}
      emptyContent={emptyContent}
      icon={ICON}
      itemHeight={45}
      dialogProps={DIALOG_PROPS}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};
