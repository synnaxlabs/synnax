// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/ranger/Select.css";

import { type ranger } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { Component } from "@/component";
import { CSS } from "@/css";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { List } from "@/list";
import { Ranger } from "@/ranger";
import { Breadcrumb } from "@/ranger/Breadcrumb";
import { type ListParams, useList } from "@/ranger/queries";
import { TimeRangeChip, type TimeRangeChipProps } from "@/ranger/TimeRangeChip";
import { HAUL_TYPE } from "@/ranger/types";
import { Select } from "@/select";
import { Tag } from "@/tag";

interface ListItemProps
  extends List.ItemProps<ranger.Key>,
    Pick<TimeRangeChipProps, "showAgo" | "showSpan"> {
  showParent?: boolean;
  showLabels?: boolean;
  starred?: boolean;
  onStar?: (starred: boolean) => void;
  onStageChange?: (stage: ranger.Stage) => void;
}

export const ListItem = ({
  itemKey,
  showParent = true,
  showLabels = true,
  starred,
  onStar,
  showAgo,
  showSpan,
  onStageChange,
  ...rest
}: ListItemProps): ReactElement | null => {
  const item = List.useItem<ranger.Key, ranger.Payload>(itemKey);
  if (item == null) return null;
  const { name, timeRange, parent, labels, stage } = item;
  const { onSelect, selected } = Select.useItemState(itemKey);

  return (
    <List.Item
      className={CSS(CSS.BE("range", "list-item"), starred && CSS.M("starred"))}
      itemKey={itemKey}
      justify="spaceBetween"
      selected={selected}
      {...rest}
    >
      <Align.Space x size="tiny" align="center">
        <Input.Checkbox
          value={selected}
          onChange={() => onSelect?.()}
          onClick={(e) => {
            e.stopPropagation();
          }}
        />
        <Align.Space x align="center" empty>
          <Ranger.SelectStage
            value={stage}
            allowNone={false}
            onChange={(v: ranger.Stage | null) => v != null && onStageChange?.(v)}
            onClick={(e) => e.stopPropagation()}
            variant="floating"
            location="bottom"
            triggerProps={{ iconOnly: true, variant: "text" }}
          />
          <Breadcrumb name={name} parent={parent} showParent={showParent} />
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
        <TimeRangeChip
          level="small"
          timeRange={timeRange}
          showAgo={showAgo}
          showSpan={showSpan}
        />
      </Align.Space>
    </List.Item>
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
