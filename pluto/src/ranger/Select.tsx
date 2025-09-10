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
import {
  type CrudeTimeRange,
  type NumericTimeRange,
  TimeRange,
  TimeStamp,
} from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Component } from "@/component";
import { type Dialog } from "@/dialog";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { List } from "@/list";
import { ListItem } from "@/ranger/ListItem";
import { type ListParams, useList } from "@/ranger/queries";
import { getStage, HAUL_TYPE, type Stage, STAGES } from "@/ranger/types";
import { Select } from "@/select";

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

export const STAGE_ICONS: Record<Stage, Icon.FC> = {
  to_do: Icon.ToDo,
  in_progress: Icon.InProgress,
  completed: Icon.Completed,
};

export const STAGE_NAMES: Record<Stage, string> = {
  to_do: "To Do",
  in_progress: "In Progress",
  completed: "Completed",
};

const DATA: Select.StaticEntry<Stage>[] = STAGES.map((s) => {
  const I = STAGE_ICONS[s];
  return { key: s, name: STAGE_NAMES[s], icon: <I /> };
});

export interface SelectStageProps
  extends Omit<Select.StaticProps<Stage>, "data" | "resourceName"> {}

export const SelectStage = (props: SelectStageProps): ReactElement => (
  <Select.Static {...props} data={DATA} resourceName="Stage" icon={<Icon.ToDo />} />
);

interface WrapNumericTimeRangeFormArgs {
  value: NumericTimeRange;
  onChange: (value: NumericTimeRange) => void;
}

interface WrapNumericTimeRangeFormReturn {
  value: Stage;
  onChange: (value: Stage) => void;
}

export const wrapNumericTimeRangeForm = ({
  value,
  onChange,
}: WrapNumericTimeRangeFormArgs): WrapNumericTimeRangeFormReturn => ({
  value: getStage(value),
  onChange: (v: Stage) => {
    if (v == null) return;
    const now = TimeStamp.now().nanoseconds;
    const tr = new TimeRange(value).makeValid().numeric;
    switch (v) {
      case "to_do":
        if (tr.end < now) tr.end = TimeStamp.MAX.nanoseconds;
        if (tr.start < now) tr.start = tr.end;
        break;
      case "in_progress":
        if (tr.start > now) tr.start = now;
        if (tr.end < now) tr.end = TimeStamp.MAX.nanoseconds;
        break;
      case "completed":
        if (tr.end > now) tr.end = now;
        if (tr.start > tr.end) tr.start = tr.end;
        break;
    }
    onChange(tr);
  },
});

export interface StageIconProps extends Icon.IconProps {
  timeRange: CrudeTimeRange;
}

export const StageIcon = ({ timeRange, ...rest }: StageIconProps): ReactElement => {
  const I = STAGE_ICONS[getStage(timeRange)];
  return <I {...rest} />;
};
