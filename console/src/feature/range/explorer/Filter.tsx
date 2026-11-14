// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/explorer/Filter.css";

import { ranger } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Flux, Ranger } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Label } from "@/platform/label";
import { View } from "@/platform/view";

/** @returns "is" for one value, else "is any of". */
const isAnyOf = (count: number): string => (count === 1 ? "is" : "is any of");

/** @returns the names, or a count when there are more than two. */
const summarize = (names: string[], noun: string): string =>
  names.length > 2 ? `${names.length} ${noun}` : names.join(", ");

const StagePicker = ({
  value,
  onChange,
}: View.Filter.PickerProps<ranger.Stage[]>): ReactElement => (
  <Select.Frame<ranger.Stage, undefined>
    multiple
    value={value ?? []}
    onChange={(next: ranger.Stage[]) => onChange(next.length === 0 ? null : next)}
  >
    <Select.Dialog surface="menu" aria-label="Stage">
      <Select.List>
        {ranger.STAGES.map((stage) => {
          const StageIcon = Ranger.STAGE_ICONS[stage];
          return (
            <Select.Item<ranger.Stage> key={stage} itemKey={stage}>
              <StageIcon />
              {Ranger.STAGE_NAMES[stage]}
            </Select.Item>
          );
        })}
      </Select.List>
    </Select.Dialog>
  </Select.Frame>
);

const STAGE: View.Filter.Property<ranger.Where, ranger.Stage[]> = {
  name: "Stage",
  icon: Icon.ToDo,
  decode: (node) =>
    View.Filter.soleField(node) === "stages" && node.stages != null
      ? node.stages
      : null,
  encode: (stages) => ({ stages }),
  Picker: StagePicker,
  Operator: ({ value }) => (
    <View.Filter.ChipOperator>{isAnyOf(value.length)}</View.Filter.ChipOperator>
  ),
  Value: ({ value }) => (
    <>
      {value.map((stage) => {
        const StageIcon = Ranger.STAGE_ICONS[stage];
        return <StageIcon key={stage} />;
      })}
      {summarize(
        value.map((stage) => Ranger.STAGE_NAMES[stage]),
        "stages",
      )}
    </>
  ),
};

/** A span matches the ranges that overlap the span that ends now. */
type Window = TimeSpan | TimeRange;

interface TimePreset {
  key: string;
  name: string;
  span: TimeSpan;
}

const TIME_PRESETS: TimePreset[] = [
  { key: "day", name: "Last 24 hours", span: TimeSpan.days(1) },
  { key: "week", name: "Last 7 days", span: TimeSpan.days(7) },
  { key: "month", name: "Last 30 days", span: TimeSpan.days(30) },
  { key: "quarter", name: "Last 90 days", span: TimeSpan.days(90) },
];

const findPreset = (span: TimeSpan): TimePreset | undefined =>
  TIME_PRESETS.find((p) => p.span.equals(span));

const TimePicker = ({
  value,
  onChange,
}: View.Filter.PickerProps<Window>): ReactElement => {
  const preset = value instanceof TimeSpan ? findPreset(value)?.key : undefined;
  const now = useNow(() => null);
  const range =
    value instanceof TimeRange
      ? value
      : new TimeRange(now.sub(value ?? TimeSpan.days(7)), now);
  return (
    <Select.Frame<string, undefined>
      value={preset}
      onChange={(key: string | null) =>
        onChange(TIME_PRESETS.find((p) => p.key === key)?.span ?? null)
      }
      allowNone
    >
      <Select.Dialog surface="menu" aria-label="Time">
        <Select.List>
          {TIME_PRESETS.map(({ key, name }) => (
            <Select.Item<string> key={key} itemKey={key}>
              {name}
            </Select.Item>
          ))}
        </Select.List>
        <Flex.Box y gap="small" className={CSS.BE("range-filter", "custom")}>
          <Text.Text level="small" color={9}>
            Custom range
          </Text.Text>
          <Input.Item x label="From" className={CSS.BE("range-filter", "bound")}>
            <Input.DateTime
              value={value == null ? undefined : range.start.nanoseconds}
              onChange={(start) =>
                onChange(new TimeRange(start, range.end).makeValid())
              }
              bound="start"
              resolution={TimeSpan.MINUTE}
              placeholder="Set a start time"
              aria-label="From"
            />
          </Input.Item>
          <Input.Item x label="To" className={CSS.BE("range-filter", "bound")}>
            <Input.DateTime
              value={value == null ? undefined : range.end.nanoseconds}
              onChange={(end) => onChange(new TimeRange(range.start, end).makeValid())}
              bound="end"
              resolution={TimeSpan.MINUTE}
              placeholder="Set an end time"
              aria-label="To"
            />
          </Input.Item>
        </Flex.Box>
      </Select.Dialog>
    </Select.Frame>
  );
};

const formatWindow = (window: Window): string => {
  if (window instanceof TimeSpan)
    return findPreset(window)?.name.toLowerCase() ?? `the last ${window.toString()}`;
  return window.toPrettyString();
};

const decodeWindow = (node: ranger.Where): Window | null => {
  const field = View.Filter.soleField(node);
  if (field === "recent" && node.recent != null) return new TimeSpan(node.recent);
  if (field === "overlaps" && node.overlaps != null)
    return new TimeRange(node.overlaps);
  return null;
};

const TIME: View.Filter.Property<ranger.Where, Window> = {
  name: "Time",
  icon: Icon.Time,
  decode: decodeWindow,
  encode: (window) =>
    window instanceof TimeSpan ? { recent: window } : { overlaps: window },
  Picker: TimePicker,
  Operator: () => <View.Filter.ChipOperator>overlaps</View.Filter.ChipOperator>,
  Value: ({ value }) => <>{formatWindow(value)}</>,
};

const ParentItem = ({
  itemKey,
  ...rest
}: List.ItemRenderProps<ranger.Key>): ReactElement | null => {
  const item = List.useItem<ranger.Key, ranger.Range>(itemKey);
  if (item == null) return null;
  return (
    <Select.Item<ranger.Key> itemKey={itemKey} {...rest}>
      {/* The name alone names the option. */}
      <Ranger.StageIcon timeRange={item.timeRange} aria-hidden />
      {item.name}
    </Select.Item>
  );
};

const parentItem = Component.renderProp(ParentItem);

const ParentPicker = ({
  value,
  onChange,
}: View.Filter.PickerProps<ranger.Key[]>): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (client) => client.ranges,
    { name: "ranges" },
  );
  return (
    <Select.Frame<ranger.Key, ranger.Range | undefined>
      multiple
      value={value ?? []}
      onChange={(next: ranger.Key[]) => onChange(next.length === 0 ? null : next)}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={more}
    >
      <Select.Dialog surface="menu" aria-label="Parent">
        <Select.Body<ranger.Key> resourceName="range" onSearch={search} status={status}>
          {parentItem}
        </Select.Body>
      </Select.Dialog>
    </Select.Frame>
  );
};

const ParentValue = ({
  value,
}: View.Filter.ConditionProps<ranger.Key[]>): ReactElement => {
  const { data: ranges = [] } = Ranger.useResultMultiple({ keys: value });
  return (
    <>
      {summarize(
        ranges.map(({ name }) => name),
        "ranges",
      )}
    </>
  );
};

const PARENT: View.Filter.Property<ranger.Where, ranger.Key[]> = {
  name: "Parent",
  icon: Icon.Range,
  decode: (node) =>
    View.Filter.soleField(node) === "parents" && node.parents != null
      ? node.parents
      : null,
  encode: (parents) => ({ parents }),
  Picker: ParentPicker,
  Operator: ({ value }) => (
    <View.Filter.ChipOperator>{isAnyOf(value.length)}</View.Filter.ChipOperator>
  ),
  Value: ParentValue,
};

/** The properties that the range explorer filters by. */
export const PROPERTIES: View.Filter.Properties<ranger.Where> = {
  labels: Label.Filter.createProperty({ unlabeled: true }),
  stage: STAGE,
  time: TIME,
  parent: PARENT,
};
