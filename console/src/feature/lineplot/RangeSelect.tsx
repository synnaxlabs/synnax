// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/lineplot/RangeSelect.css";

import { lineplot, NotFoundError, ranger } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Flux, LinePlot, Ranger } from "@synnaxlabs/pluto";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";

import { CSS } from "@/platform/css";
import { LinePlot as PlatformLinePlot } from "@/platform/lineplot";
import { Range } from "@/platform/range";

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

// The rolling window's tag is keyed like the window's lines.
const ROLLING_KEY = lineplot.ROLLING_LINE_RANGE;

// A selection reports only keys, so a typed window's option key carries its span.
const DRAFT_PREFIX = `${ROLLING_KEY}:`;

const liveIcon = <Icon.Dynamic className={CSS.BE("range-select", "live-icon")} />;

interface RollingOptionProps {
  /** The axis's rolling window, in nanoseconds. */
  span?: number;
}

const RollingOption = ({ span }: RollingOptionProps): ReactElement | null => {
  const term = Select.useSearchTerm();
  const typed = TimeSpan.parse(term);
  if (typed == null || typed.isZero) return null;
  // Typing the held window offers the window itself, so the option can deselect it.
  const itemKey =
    span != null && typed.equals(span)
      ? ROLLING_KEY
      : `${DRAFT_PREFIX}${typed.valueOf()}`;
  // The search hides fixed items whose text lacks the term, so the option echoes it.
  return (
    <Select.Item itemKey={itemKey}>
      <Text.Text>
        {liveIcon}
        {term.trim()}
      </Text.Text>
    </Select.Item>
  );
};

const FavoriteItem = Component.renderProp((props: List.ItemProps<string>) => {
  const range = Range.useResolve(props.itemKey);
  const { data: parent } = Ranger.useResultParent({
    id: ranger.ontologyID(props.itemKey),
  });
  if (range == null) return null;
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

interface RollingTagProps {
  span: number;
  onChange: (span: number) => void;
}

const RollingTag = ({ span, onChange }: RollingTagProps): ReactElement => {
  const { onSelect } = Select.useItemState(ROLLING_KEY);
  return (
    <Tag.Tag
      icon={liveIcon}
      onClose={onSelect}
      level="small"
      size="small"
      className={CSS.BE("range-select", "editable")}
    >
      <Input.TimeSpan value={span} onChange={onChange} variant="text" size="small" />
    </Tag.Tag>
  );
};

interface PersistedTagProps {
  range: lineplot.PersistedRange;
}

const PersistedTag = ({ range: { key } }: PersistedTagProps): ReactElement => {
  const { onSelect } = Select.useItemState(key);
  const result = Ranger.useResult({ key });
  let name = "";
  let tooltip: string | undefined;
  if (result.variant === "success") name = result.data.name;
  else if (result.variant === "error") {
    const { error } = result.status.details;
    const deleted =
      Flux.DeletedError.matches(error) || NotFoundError.matches(error.cause);
    name = deleted ? "Deleted range" : "Unavailable range";
    tooltip = result.status.message;
  }
  return (
    <Tag.Tag
      icon={result.variant === "loading" ? <Icon.Loading /> : <Icon.Range />}
      onClose={onSelect}
      level="small"
      size="small"
      textColor={result.variant === "error" ? 8 : undefined}
      tooltip={tooltip}
    >
      {name}
    </Tag.Tag>
  );
};

interface StaticTagProps {
  range: lineplot.StaticRange;
  onChange: (range: lineplot.StaticRange) => void;
}

const StaticTag = ({ range, onChange }: StaticTagProps): ReactElement => {
  const { onSelect } = Select.useItemState(range.key);
  const start = Number(range.start);
  const end = Number(range.end);
  return (
    <Tag.Tag
      icon={<Icon.Range />}
      onClose={onSelect}
      level="small"
      size="small"
      tooltip={range.name}
      className={CSS.BE("range-select", "editable")}
    >
      <Input.DateTime
        value={start}
        onChange={(next) => onChange({ ...range, start: String(next) })}
        variant="text"
        size="small"
        bound="start"
        anchors={{ end }}
      />
      <Icon.Arrow.Right />
      <Input.DateTime
        value={end}
        onChange={(next) => onChange({ ...range, end: String(next) })}
        variant="text"
        size="small"
        bound="end"
        anchors={{ start }}
        sharedDay={start}
      />
    </Tag.Tag>
  );
};

interface RollingRowProps {
  span?: number;
  onChange: (span?: number) => void;
}

// A frame of its own, so its toggles stay out of the list's arrow keys and search.
const RollingRow = ({ span, onChange }: RollingRowProps): ReactElement => {
  const handleChange = useCallback(
    (key: string | null) => onChange(key == null ? undefined : Number(key)),
    [onChange],
  );
  return (
    <Flex.Box x align="center" gap="medium" className={CSS.BE("range-select", "live")}>
      <Text.Text className={CSS.BE("range-select", "live-label")}>
        <Icon.Dynamic />
        Live
      </Text.Text>
      <Select.Buttons
        allowNone
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
    No favorite ranges found
  </Status.Summary>
);

const sameKeys = (a: lineplot.Range[], b: lineplot.Range[]): boolean =>
  a.length === b.length && a.every((r, i) => r.key === b[i].key);

export interface XAxisRangeSelectProps extends Omit<
  Input.ItemProps,
  "label" | "onChange" | "children"
> {
  axisKey: lineplot.XAxisKey;
}

/**
 * Selects what the x-axis plots: at most one rolling window, favorite ranges, and
 * static windows. Each tag edits its window in place.
 */
export const XAxisRangeSelect = ({
  axisKey,
  ...rest
}: XAxisRangeSelectProps): ReactElement => {
  const { rolling, ranges } = LinePlot.useXAxisRanges({ axisKey });
  const dispatch = LinePlot.useSingleDispatch();
  const favorites = Range.useResolveMultiple();
  const { data, retrieve } = List.useStaticData<string>({ data: favorites });
  const { fetchMore, search } = List.usePager({ retrieve });
  const value = useMemo(
    () => [...(rolling == null ? [] : [ROLLING_KEY]), ...ranges.map(({ key }) => key)],
    [rolling, ranges],
  );
  const setRolling = useCallback(
    (span?: number) => dispatch(lineplot.setRolling({ axisKey, span })),
    [dispatch, axisKey],
  );
  const setRange = useCallback(
    (range: lineplot.Range) => dispatch(lineplot.setRange({ axisKey, range })),
    [dispatch, axisKey],
  );
  const handleChange = useCallback(
    (keys: string[]) => {
      const actions: lineplot.Action[] = [];
      const draft = keys.find((k) => k.startsWith(DRAFT_PREFIX));
      if (draft != null)
        actions.push(
          lineplot.setRolling({
            axisKey,
            span: Number(draft.slice(DRAFT_PREFIX.length)),
          }),
        );
      else if (rolling != null && !keys.includes(ROLLING_KEY))
        actions.push(lineplot.setRolling({ axisKey, span: undefined }));
      const held = new Map(ranges.map((r) => [r.key, r]));
      const next = keys.flatMap((key): lineplot.Range[] => {
        const range = held.get(key);
        if (range != null) return [range];
        const favorite = favorites.find((f) => f.key === key);
        return favorite == null ? [] : [PlatformLinePlot.fromSession(favorite)];
      });
      if (!sameKeys(next, ranges))
        actions.push(lineplot.setRanges({ axisKey, ranges: next }));
      if (actions.length > 0) dispatch(actions);
    },
    [dispatch, axisKey, rolling, ranges, favorites],
  );
  const renderTag = useCallback(
    ({ itemKey }: Select.MultipleTagProps<string>) => {
      if (itemKey === ROLLING_KEY && rolling != null)
        return <RollingTag key={itemKey} span={rolling} onChange={setRolling} />;
      const range = ranges.find(({ key }) => key === itemKey);
      if (range == null) return null;
      return range.variant === "persisted" ? (
        <PersistedTag key={itemKey} range={range} />
      ) : (
        <StaticTag key={itemKey} range={range} onChange={setRange} />
      );
    },
    [rolling, ranges, setRolling, setRange],
  );
  return (
    <Input.Item x label="Ranges" {...rest}>
      <Dialog.Frame variant="connected">
        <Select.Frame<string>
          multiple
          value={value}
          onChange={handleChange}
          data={data}
          onFetchMore={fetchMore}
          initialHover={0}
          virtual
        >
          <Select.MultipleTrigger<string>
            icon={<Icon.Range />}
            placeholder="Select ranges"
            aria-label="Ranges"
          >
            {renderTag}
          </Select.MultipleTrigger>
          <Select.Dialog>
            <Select.Search
              placeholder="Search favorites or type a duration"
              onSearch={search}
            />
            <RollingRow span={rolling} onChange={setRolling} />
            <Select.List bordered borderColor={6} grow rounded full="x">
              <RollingOption span={rolling} />
              <Select.Items<string> emptyContent={emptyContent}>
                {FavoriteItem}
              </Select.Items>
            </Select.List>
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>
    </Input.Item>
  );
};
