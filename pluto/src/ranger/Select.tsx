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
import { Button } from "@/button";
import { Component } from "@/component";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { List } from "@/list";
import { Ranger } from "@/ranger";
import { type ListParams, useList } from "@/ranger/queries";
import { TimeRangeChip } from "@/ranger/TimeRangeChip";
import { HAUL_TYPE } from "@/ranger/types";
import { Select } from "@/select";
import { Tag } from "@/tag";

interface ListItemProps extends List.ItemProps<ranger.Key> {
  showParent?: boolean;
  showLabels?: boolean;
}

export const ListItem = ({
  itemKey,
  showParent = true,
  showLabels = true,
  ...rest
}: ListItemProps): ReactElement | null => {
  const item = List.useItem<ranger.Key, ranger.Payload>(itemKey);
  if (item == null) return null;
  const { name, timeRange, parent, labels, stage } = item;
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

  const Icon = Ranger.STAGE_ICONS[stage];
  return (
    <Select.ListItem itemKey={itemKey} justify="spaceBetween" {...rest}>
      <Align.Space x align="center" size="small">
        <Button.Icon>
          <Icon />
        </Button.Icon>
        <Align.Space y size="small">
          <Breadcrumb.Breadcrumb>{breadcrumbSegments}</Breadcrumb.Breadcrumb>
        </Align.Space>
      </Align.Space>
      <Align.Space x>
        {showLabels && (
          <Tag.Tags>
            {labels?.map(({ key, name, color }) => (
              <Tag.Tag key={key} color={color} size="small" shade={9}>
                {name}
              </Tag.Tag>
            ))}
          </Tag.Tags>
        )}
        <TimeRangeChip level="p" timeRange={timeRange} />
      </Align.Space>
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
  const { fetchMore, search } = List.usePager({ retrieve });
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
      icon={<Icon.Range />}
      itemHeight={56}
      dialogProps={{ style: { width: 800 } }}
      {...rest}
    >
      {listItemRenderProp}
    </Select.Single>
  );
};

export const STAGE_ICONS: Record<ranger.Stage, Icon.FC> = {
  to_do: Icon.ToDo,
  in_progress: Icon.InProgress,
  completed: Icon.Completed,
};

const DATA: Select.SimplyEntry<ranger.Stage>[] = [
  { key: "to_do", name: "To Do", icon: <STAGE_ICONS.to_do /> },
  { key: "in_progress", name: "In Progress", icon: <STAGE_ICONS.in_progress /> },
  { key: "completed", name: "Completed", icon: <STAGE_ICONS.completed /> },
];

export interface SelectStageProps
  extends Omit<Select.SimpleProps<ranger.Stage>, "data" | "resourceName"> {}

export const SelectStage = (props: SelectStageProps): ReactElement => (
  <Select.Simple {...props} data={DATA} resourceName="Stage" icon={<Icon.ToDo />} />
);
