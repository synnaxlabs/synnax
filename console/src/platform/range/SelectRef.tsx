// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/range/SelectRef.css";

import { NotFoundError, ranger } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Flux, Ranger } from "@synnaxlabs/pluto";
import { type telem, TimeRange, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";

import { CSS } from "@/platform/css";
import { setBound, toRef } from "@/platform/range/ref";
import {
  type Resolved,
  useResolve,
  useResolveMultiple,
} from "@/platform/range/resolve";
import { type Session } from "@/session";

/** The rolling windows the dialog offers in one click. */
const ROLLING_PRESETS = [
  TimeSpan.seconds(30),
  TimeSpan.minutes(1),
  TimeSpan.minutes(5),
  TimeSpan.minutes(15),
  TimeSpan.minutes(30),
  TimeSpan.hours(1),
  TimeSpan.hours(6),
  TimeSpan.hours(12),
  TimeSpan.days(1),
  TimeSpan.days(7),
  TimeSpan.days(30),
].map(Number);

const ROLLING_KEY = "rolling";
const STATIC_KEY = "static";
const TYPED_KEY = "typed-rolling";

const liveIcon = <Icon.Dynamic className={CSS.BE("range-ref", "live-icon")} />;

type Listed = Exclude<Resolved, Session.Range.DynamicState>;

const rolling = (timeSpan: number): ranger.Ref => ({
  variant: "window",
  window: { variant: "dynamic", timeSpan },
});

interface RollingOptionProps {
  span?: number;
  onChange: (span: number) => void;
}

const RollingOption = ({ span, onChange }: RollingOptionProps): ReactElement | null => {
  const term = Select.useSearchTerm();
  const { close } = Dialog.useContext();
  const typed = TimeSpan.parse(term);
  if (typed == null || typed.isZero) return null;
  // The search hides fixed items whose text lacks the term, so the option echoes it.
  return (
    <Select.Item
      itemKey={TYPED_KEY}
      selected={span != null && typed.equals(span)}
      onSelect={() => {
        onChange(Number(typed));
        close();
      }}
    >
      <Text.Text>
        {liveIcon}
        {term.trim()}
      </Text.Text>
    </Select.Item>
  );
};

const RangeItem = Component.renderProp((props: List.ItemProps<string>) => {
  const inSession = useResolve(props.itemKey);
  const listed = List.useItem<string, Listed>(props.itemKey);
  const range = inSession ?? listed;
  const { data: parent } = Ranger.useResultParent({
    id: ranger.ontologyID(props.itemKey),
  });
  if (range == null || range.variant === "dynamic") return null;
  return (
    <Select.Item {...props} justify="between">
      <Ranger.Breadcrumb
        name={range.name}
        parent={parent}
        timeRange={range.timeRange}
      />
      <Telem.Text.TimeRange level="small">{range.timeRange}</Telem.Text.TimeRange>
    </Select.Item>
  );
});

// The labels render as fragments, since the trigger button is already a text element.
const SavedLabel = ({ rangeKey }: { rangeKey: string }): ReactElement => {
  const result = Ranger.useResult({ key: rangeKey });
  if (result.variant === "success")
    return (
      <>
        <Icon.Range />
        {result.data.name}
      </>
    );
  if (result.variant !== "error") return <Icon.Loading />;
  const { error } = result.status.details;
  const deleted =
    Flux.DeletedError.matches(error) || NotFoundError.matches(error.cause);
  return (
    <>
      <Icon.Range />
      {deleted ? "Deleted range" : "Unavailable range"}
    </>
  );
};

const formatFixed = ({ timeRange }: telem.StaticWindow): string => {
  const tr = new TimeRange(BigInt(timeRange.start), BigInt(timeRange.end));
  const endFormat = tr.span.lessThan(TimeSpan.DAY) ? "time" : "dateTime";
  return `${tr.start.toString("dateTime", "local")} → ${tr.end.toString(endFormat, "local")}`;
};

const Label = ({ value }: { value: ranger.Ref }): ReactElement => {
  if (value.variant === "key") return <SavedLabel rangeKey={value.key} />;
  if (value.window.variant === "dynamic")
    return (
      <>
        {liveIcon}
        {`Rolling ${new TimeSpan(value.window.timeSpan).toString()}`}
      </>
    );
  return (
    <>
      <Icon.Range />
      {formatFixed(value.window)}
    </>
  );
};

interface FixedRowProps {
  window: telem.StaticWindow;
  onChange: (window: telem.StaticWindow) => void;
}

const FixedRow = ({ window, onChange }: FixedRowProps): ReactElement => {
  const start = Number(window.timeRange.start);
  const end = Number(window.timeRange.end);
  const set = (bound: "start" | "end", value: number): void =>
    onChange(setBound(window, bound, value));
  return (
    <Flex.Box x align="center" gap="medium" className={CSS.BE("range-ref", "row")}>
      <Text.Text className={CSS.BE("range-ref", "row-label")}>
        <Icon.Range />
        Fixed
      </Text.Text>
      <Input.DateTime
        value={start}
        onChange={(next) => set("start", next)}
        size="small"
        bound="start"
        anchors={{ end }}
      />
      <Icon.Arrow.Right />
      <Input.DateTime
        value={end}
        onChange={(next) => set("end", next)}
        size="small"
        bound="end"
        anchors={{ start }}
        sharedDay={start}
      />
    </Flex.Box>
  );
};

interface RollingRowProps {
  span?: number;
  onChange: (span: number) => void;
}

// A frame of its own, so its toggles stay out of the list's arrow keys and search.
const RollingRow = ({ span, onChange }: RollingRowProps): ReactElement => {
  const { close } = Dialog.useContext();
  const handleChange = useCallback(
    (key: string | null) => {
      if (key == null) return;
      onChange(Number(key));
      close();
    },
    [onChange, close],
  );
  return (
    <Flex.Box x align="center" gap="medium" className={CSS.BE("range-ref", "row")}>
      <Text.Text className={CSS.BE("range-ref", "live-label")}>
        <Icon.Dynamic />
        Live
      </Text.Text>
      <Select.Buttons
        value={span == null ? undefined : String(span)}
        onChange={handleChange}
        enableTriggers={false}
        wrap
      >
        {ROLLING_PRESETS.map((preset) => (
          <Select.Item key={String(preset)} itemKey={String(preset)}>
            {new TimeSpan(preset).toString()}
          </Select.Item>
        ))}
      </Select.Buttons>
    </Flex.Box>
  );
};

const emptyContent = (
  <Status.Summary center variant="disabled">
    No ranges found
  </Status.Summary>
);

const selectedKey = (ref: ranger.Ref, listed: Listed[]): string => {
  if (ref.variant === "key") return ref.key;
  if (ref.window.variant === "dynamic") return ROLLING_KEY;
  const { start, end } = ref.window.timeRange;
  const match = listed.find((range) => {
    if (range.variant !== "static") return false;
    const r = toRef(range);
    return (
      r.variant === "window" &&
      r.window.variant === "static" &&
      r.window.timeRange.start === start &&
      r.window.timeRange.end === end
    );
  });
  return match?.key ?? STATIC_KEY;
};

export interface SelectRefInputItemProps extends Omit<
  Input.ItemProps,
  "label" | "onChange" | "children"
> {
  value: ranger.Ref;
  onChange: (value: ranger.Ref) => void;
  disabled?: boolean;
}

/**
 * Selects one range: a rolling window, or a range the session holds. The dialog also
 * takes a typed duration and edits a fixed window's bounds.
 */
export const SelectRefInputItem = ({
  value,
  onChange,
  disabled,
  ...rest
}: SelectRefInputItemProps): ReactElement => {
  const sessionRanges = useResolveMultiple();
  const inSession = useMemo(
    () => sessionRanges.filter((r): r is Listed => r.variant !== "dynamic"),
    [sessionRanges],
  );
  // A saved range the session lacks, such as one from a shared visualization, still
  // needs an entry for the list to highlight it.
  const outside =
    value.variant === "key" && !inSession.some(({ key }) => key === value.key);
  const { data: outsideRange } = Ranger.useResult(
    outside && value.variant === "key" ? { key: value.key } : null,
  );
  const listed = useMemo(
    (): Listed[] =>
      !outside || outsideRange == null
        ? inSession
        : [
            ...inSession,
            {
              variant: "persisted",
              key: outsideRange.key,
              name: outsideRange.name,
              timeRange: outsideRange.timeRange.numeric,
            },
          ],
    [inSession, outside, outsideRange],
  );
  const { data, getItem, retrieve } = List.useStaticData<string, Listed>({
    data: listed,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  const selected = useMemo(() => selectedKey(value, listed), [value, listed]);
  const span =
    value.variant === "window" && value.window.variant === "dynamic"
      ? value.window.timeSpan
      : undefined;
  const setRolling = useCallback(
    (timeSpan: number) => onChange(rolling(timeSpan)),
    [onChange],
  );
  const setWindow = useCallback(
    (window: telem.Window) => onChange({ variant: "window", window }),
    [onChange],
  );
  const handleChange = useCallback(
    (key: string) => {
      const range = listed.find((r) => r.key === key);
      if (range != null) onChange(toRef(range));
    },
    [listed, onChange],
  );
  return (
    <Input.Item label="Range" {...rest}>
      <Dialog.Frame variant="connected" className={CSS.B("range-ref")}>
        <Select.Frame<string>
          allowNone={false}
          closeDialogOnSelect
          value={selected}
          onChange={handleChange}
          data={data}
          getItem={getItem}
          onFetchMore={fetchMore}
          initialHover={0}
          virtual
        >
          <Dialog.Trigger
            variant="outlined"
            gap="small"
            aria-label="Range"
            disabled={disabled}
            full="x"
          >
            <Label value={value} />
          </Dialog.Trigger>
          <Select.Dialog className={CSS.BE("range-ref", "dialog")}>
            <Select.Search
              placeholder="Search ranges or type a duration"
              onSearch={search}
            />
            <RollingRow span={span} onChange={setRolling} />
            {value.variant === "window" && value.window.variant === "static" && (
              <FixedRow window={value.window} onChange={setWindow} />
            )}
            <Select.List bordered borderColor={6} grow rounded full="x">
              <RollingOption span={span} onChange={setRolling} />
              <Select.Items<string> emptyContent={emptyContent}>
                {RangeItem}
              </Select.Items>
            </Select.List>
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>
    </Input.Item>
  );
};
