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

import { Breadcrumb } from "@/breadcrumb";
import { Component } from "@/component";
import { type Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { List } from "@/list";
import { type ListQuery, useList } from "@/ranger/queries";
import { HAUL_TYPE } from "@/ranger/types";
import { Select } from "@/select";
import { Tag } from "@/tag";
import { Telem } from "@/telem";

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
    return (
      <Select.ListItem itemKey={itemKey} justify="between" {...rest}>
        <Breadcrumb.Breadcrumb>
          <Breadcrumb.Segment weight={450} color={10}>
            {name}
          </Breadcrumb.Segment>
          {parent != null && showParent && (
            <Breadcrumb.Segment weight={400} color={8}>
              {parent.name}
            </Breadcrumb.Segment>
          )}
        </Breadcrumb.Breadcrumb>
        <Flex.Box x>
          <Telem.Text.TimeRange level="small">{timeRange}</Telem.Text.TimeRange>
          {showLabels && (
            <Tag.Tags variant="text">
              {labels?.map((l) => (
                <Tag.Tag key={l.key} color={l.color} size="small">
                  {l.name}
                </Tag.Tag>
              ))}
            </Tag.Tags>
          )}
        </Flex.Box>
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
    Flux.UseListParams<ListQuery, ranger.Key, ranger.Payload> {}

const ICON = <Icon.Range />;

export const SelectMultiple = ({
  onChange,
  value,
  emptyContent,
  filter,
  initialQuery,
  ...rest
}: SelectMultipleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe, status } = useList({
    filter,
    initialQuery,
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
    Flux.UseListParams<ListQuery, ranger.Key, ranger.Payload> {}

const DIALOG_PROPS: Dialog.DialogProps = {
  style: { width: 800 },
};

export const SelectSingle = ({
  onChange,
  value,
  filter,
  allowNone,
  emptyContent,
  initialQuery,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, subscribe, getItem, status } = useList({
    filter,
    initialQuery,
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
