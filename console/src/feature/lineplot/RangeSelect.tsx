// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/lineplot/RangeSelect.css";

import { lineplot, NotFoundError, type ranger } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { CSS as PCSS } from "@synnaxlabs/lyra/css";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Haul } from "@synnaxlabs/lyra/haul";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import type { position } from "@synnaxlabs/lyra/position";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Flux, LinePlot, Ranger } from "@synnaxlabs/pluto";
import {
  type DynamicWindow,
  dynamicWindow,
  resolveWindow,
  type StaticWindow,
  staticWindow,
  type TimeRange,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useState } from "react";

import { CSS } from "@/platform/css";
import { Session } from "@/session";

const ROLLING_PRESETS = Session.Range.BUILT_IN.map(({ span }) => span);

const liveIcon = <Icon.Dynamic className={CSS.BE("range-select", "live-icon")} />;

interface RollingRef extends ranger.WindowRef {
  window: DynamicWindow;
}

const isRolling = (ref: ranger.Ref): ref is RollingRef =>
  ref.variant === "window" && ref.window.variant === "dynamic";

const rollingRef = (span: TimeSpan): RollingRef => ({
  variant: "window",
  window: dynamicWindow(span),
});

// Two session ranges can show the same window, and a plot draws each reference once.
const dedupe = (refs: ranger.Ref[]): ranger.Ref[] =>
  refs.filter(
    (r, i) => refs.findIndex((other) => Session.Range.refsEqual(other, r)) === i,
  );

// A window no session range shows is keyed by the window itself.
const windowKey = ({ window }: ranger.WindowRef): string =>
  window.variant === "dynamic"
    ? `rolling/${window.timeSpan}`
    : `static/${window.timeRange.start}/${window.timeRange.end}`;

const RangeItem = (props: List.ItemProps<ranger.Key>): ReactElement | null => {
  const range = List.useItem<ranger.Key, ranger.Range>(props.itemKey);
  const { data: parent } = Ranger.useResultParent({ key: props.itemKey });
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
};

const rangeItem = Component.renderProp(RangeItem);

const NEWEST: ranger.Order = { field: "time_range_start", descending: true };

interface RollingTagProps {
  onRemove: () => void;
  window: DynamicWindow;
}

const RollingTag = ({ onRemove, window }: RollingTagProps): ReactElement => {
  return (
    <Tag.Tag icon={liveIcon} onClose={onRemove} level="small" size="small">
      {new TimeSpan(window.timeSpan).toString()}
    </Tag.Tag>
  );
};

interface KeyTagProps {
  onRemove: () => void;
  range: ranger.KeyRef;
}

const KeyTag = ({ onRemove, range: { key } }: KeyTagProps): ReactElement => {
  const result = Ranger.useResult({ key });
  let name = "";
  let tooltip: string | undefined;
  let icon: Icon.ReactElement = <Icon.Loading />;
  if (result.variant === "success") {
    name = result.data.name;
    icon = <Ranger.StageIcon timeRange={result.data.timeRange} />;
  } else if (result.variant === "error") {
    icon = <Icon.Range />;
    const { error } = result.status.details;
    const deleted =
      Flux.DeletedError.matches(error) || NotFoundError.matches(error.cause);
    name = deleted ? "Deleted range" : "Unavailable range";
    tooltip = result.status.message;
  }
  return (
    <Tag.Tag
      icon={icon}
      onClose={onRemove}
      level="small"
      size="small"
      textColor={result.variant === "error" ? 8 : undefined}
      tooltip={tooltip}
    >
      {name}
    </Tag.Tag>
  );
};

// About ten steps of the resolution fit in the span, so a 15 minute window shows
// minutes and a 30 second window shows seconds.
const RESOLUTIONS = [
  TimeSpan.HOUR,
  TimeSpan.MINUTE,
  TimeSpan.SECOND,
  TimeSpan.MILLISECOND,
];

const resolutionOf = (span: TimeSpan): TimeSpan | undefined =>
  RESOLUTIONS.find((r) => r.valueOf() * 10n <= span.valueOf());

const formatWindow = (timeRange: TimeRange, now: TimeStamp): string => {
  const { start, end } = timeRange;
  const resolution = resolutionOf(timeRange.span);
  const startDay = start.isSameDay(now, "local")
    ? ""
    : `${Telem.Text.describeDay(start, now)} `;
  const endDay = end.isSameDay(start, "local")
    ? ""
    : `${Telem.Text.describeDay(end, now)} `;
  const startTime = Telem.Text.formatTime(start, resolution);
  const endTime = Telem.Text.formatTime(end, resolution);
  return `${startDay}${startTime} → ${endDay}${endTime}`;
};

const formatExact = ({ start, end }: TimeRange): string =>
  `${start.toPreciseString("local")} → ${end.toPreciseString("local")}`;

interface StaticTagProps {
  onRemove: () => void;
  window: StaticWindow;
}

const StaticTag = ({ onRemove, window }: StaticTagProps): ReactElement => {
  const timeRange = resolveWindow(window, TimeStamp.now());
  // The day prefix changes at midnight.
  const now = useNow((now) => Telem.Text.nextMidnight(now));
  return (
    <Tag.Tag
      icon={<Ranger.StageIcon timeRange={timeRange} />}
      onClose={onRemove}
      level="small"
      size="small"
      tooltip={formatExact(timeRange)}
    >
      {formatWindow(timeRange, now)}
    </Tag.Tag>
  );
};

interface LiveRowProps {
  span?: TimeSpan;
  onChange: (span?: TimeSpan) => void;
}

// The Off chip's key: a zero-length window is no window.
const OFF = 0;

// A frame of its own, so its toggles stay out of the list's arrow keys and search.
const LiveRow = ({ span, onChange }: LiveRowProps): ReactElement => (
  <Flex.Box x align="center" gap="medium" className={CSS.BE("range-select", "live")}>
    <Text.Text className={CSS.BE("range-select", "live-label")}>
      <Icon.Dynamic />
      Live
    </Text.Text>
    <Select.Buttons
      allowNone
      enableTriggers={false}
      value={span == null ? OFF : Number(span.valueOf())}
      onChange={(key: number | null) =>
        onChange(key == null || key === OFF ? undefined : new TimeSpan(key))
      }
      size="small"
      wrap
    >
      <Select.Item itemKey={OFF} size="small">
        <Icon.None />
        Off
      </Select.Item>
      {ROLLING_PRESETS.map((preset) => (
        <Select.Item key={preset} itemKey={preset} size="small">
          {new TimeSpan(preset).toString()}
        </Select.Item>
      ))}
    </Select.Buttons>
  </Flex.Box>
);

const RECENT_RE = /^(?:the\s+)?(?:last|past)\s+(?:an?\s+)?(.+)$/i;
const SHORT_UNIT_RE = /^(?:hour|minute|second)$/i;

// `last 2 hours` and `past hour` read as their duration. Only a short unit drops its
// number, because `last week` names the calendar week.
const parseSpan = (term: string): TimeSpan | null => {
  const recent = RECENT_RE.exec(term.trim());
  if (recent == null) return TimeSpan.parse(term);
  const [, length] = recent;
  return TimeSpan.parse(SHORT_UNIT_RE.test(length) ? `1 ${length}` : length);
};

// A duration reads as a rolling window. Other text that reads as one time range, such
// as `yesterday from 1pm to 2pm`, reads as a static window.
const readWindow = (
  term: string,
  language: Input.Language | null,
  now: TimeStamp,
): Window | undefined => {
  const span = parseSpan(term);
  if (span != null) return span.isZero ? undefined : dynamicWindow(span);
  if (language == null) return undefined;
  const timeRange = Input.suggestTimeRange(term, language, now);
  return timeRange == null ? undefined : staticWindow(timeRange);
};

interface WindowSearchProps {
  onSearch: (term: string) => void;
  /** The typed window. Undefined while the search reads as no window. */
  typed?: Entry;
  onEnter: (typed: Entry) => void;
}

// Enter on the typed window's option takes it and closes the dialog. Enter on another
// option goes to the list. The wrapper packs, so the field takes the dialog's top
// corners.
const WindowSearch = ({
  onSearch,
  typed,
  onEnter,
}: WindowSearchProps): ReactElement => {
  const { close } = Dialog.useContext();
  const { getState } = Select.useContext<string>();
  return (
    <Flex.Box
      y
      pack
      onKeyDown={(e) => {
        if (e.key !== "Enter" || typed == null) return;
        const { hover } = getState();
        if (hover != null && hover !== typed.key) return;
        // Keeps the list's window-level Enter trigger from clicking a range.
        e.stopPropagation();
        onEnter(typed);
        close();
      }}
    >
      <Select.Search
        placeholder="Search ranges or type a duration or time"
        onSearch={onSearch}
      />
    </Flex.Box>
  );
};

interface WindowOptionProps {
  itemKey: string;
  term: string;
  window: Window;
}

// Its text is the term, so the search that typed it never hides it.
const WindowOption = ({ itemKey, term, window }: WindowOptionProps): ReactElement => {
  const now = useNow((now) => Telem.Text.nextMidnight(now));
  const timeRange = resolveWindow(window, now);
  const live = window.variant === "dynamic";
  return (
    <Select.Item itemKey={itemKey} textValue={term} justify="between">
      <Text.Text>
        {live ? liveIcon : <Ranger.StageIcon timeRange={timeRange} />}
        {live ? timeRange.span.toString() : formatWindow(timeRange, now)}
      </Text.Text>
      <Text.Text level="small" color={9}>
        {live ? "Live window" : "Fixed window"}
      </Text.Text>
    </Select.Item>
  );
};

const ABOVE: position.LocationPreference = {
  targetCorner: "top",
  dialogCorner: "bottom",
};

const NO_RANGES = (
  <Status.Summary center variant="disabled">
    No ranges found
  </Status.Summary>
);

interface Entry {
  key: string;
  ref: ranger.Ref;
}

export interface RangeSelectProps extends Omit<
  Input.ItemProps,
  "label" | "onChange" | "children"
> {}

/**
 * Selects what the plot shows: at most one rolling window, Core ranges, and static
 * windows. A duration typed into the search lists a live option, and a time range such
 * as `yesterday from 1pm to 2pm` lists a fixed one. Enter takes the option. A range
 * dropped on it joins the plot.
 */
export const RangeSelect = (props: RangeSelectProps): ReactElement => {
  const refs = LinePlot.useRanges();
  const states = Session.Range.useSelectMultiple();
  const dispatch = LinePlot.useSingleDispatch();
  const { data, getItem, subscribe, status, more, search } = Flux.useList(
    (c) => c.ranges,
    { name: "ranges", query: { order: NEWEST } },
  );
  // `1h ago` reads against the time it was typed, so Enter adds what the option shows.
  const [{ term, typedAt }, setTerm] = useState(() => ({
    term: "",
    typedAt: TimeStamp.now(),
  }));
  const language = Input.useLanguage();
  const entryOf = (ref: ranger.Ref): Entry => ({
    ref,
    key:
      ref.variant === "key"
        ? ref.key
        : (Session.Range.findKey(ref, states) ?? windowKey(ref)),
  });
  const entries = refs.map(entryOf);
  const typedWindow = readWindow(term, language, typedAt);
  const typed =
    typedWindow == null
      ? undefined
      : { window: typedWindow, ...entryOf({ variant: "window", window: typedWindow }) };
  const rolling = refs.find(isRolling);
  const span = rolling == null ? undefined : new TimeSpan(rolling.window.timeSpan);
  const setRanges = (ranges: ranger.Ref[]) =>
    dispatch(lineplot.setRanges({ ranges: dedupe(ranges) }));
  const dragging = Haul.useDraggingState();
  const dropProps = Haul.useDrop({
    type: "line_plot_range_select",
    canDrop: Ranger.canDropHaulItem,
    onDrop: ({ items }) => {
      const dropped = Ranger.filterHaulItems(items);
      setRanges([
        ...refs,
        ...dropped.map(({ key }): ranger.Ref => ({ variant: "key", key })),
      ]);
      return dropped;
    },
  });
  const setRolling = (next?: TimeSpan) => {
    if (next == null) return setRanges(refs.filter((r) => !isRolling(r)));
    if (rolling == null) return setRanges([...refs, rollingRef(next)]);
    setRanges(
      refs.flatMap((r) => {
        if (r === rolling) return [rollingRef(next)];
        return isRolling(r) ? [] : [r];
      }),
    );
  };
  const take = ({ ref }: Entry) => {
    if (isRolling(ref)) return setRolling(new TimeSpan(ref.window.timeSpan));
    setRanges([...refs, ref]);
  };
  const handleChange = (keys: string[]) => {
    if (
      typed != null &&
      keys.includes(typed.key) &&
      !entries.some(({ key }) => key === typed.key)
    )
      return take(typed);
    const held = new Map(entries.map(({ key, ref }) => [key, ref]));
    setRanges(
      keys.map((key): ranger.Ref => {
        const ref = held.get(key);
        if (ref != null) return ref;
        const state = states.find((s) => s.key === key);
        return state == null ? { variant: "key", key } : Session.Range.toRef(state);
      }),
    );
  };
  let empty: ReactNode = NO_RANGES;
  if (typed != null) empty = null;
  else if (status.variant === "loading")
    empty = term === "" ? <Status.Loading /> : null;
  else if (status.variant === "error")
    empty = (
      <Status.Summary center variant="error" description={status.description}>
        {status.message}
      </Status.Summary>
    );
  const renderTag = ({ itemKey, onRemove }: Select.MultipleTagProps<string>) => {
    const entry = entries.find(({ key }) => key === itemKey);
    if (entry == null) return null;
    const { ref } = entry;
    if (ref.variant === "key")
      return <KeyTag key={itemKey} onRemove={onRemove} range={ref} />;
    if (ref.window.variant === "dynamic")
      return <RollingTag key={itemKey} onRemove={onRemove} window={ref.window} />;
    return <StaticTag key={itemKey} onRemove={onRemove} window={ref.window} />;
  };
  return (
    <Input.Item x label="Ranges" {...props} {...dropProps}>
      <Dialog.Frame variant="floating" location={ABOVE}>
        <Select.Frame<string, ranger.Range | undefined>
          multiple
          value={entries.map(({ key }) => key)}
          onChange={handleChange}
          data={data}
          getItem={getItem}
          subscribe={subscribe}
          onFetchMore={more}
          initialHover={0}
          virtual
          selectedFirst
        >
          <Select.MultipleTrigger<string>
            icon={<Icon.Range />}
            placeholder="Select ranges"
            aria-label="Ranges"
            className={PCSS.cls(PCSS.dropRegion(Ranger.canDropHaulItem(dragging)))}
          >
            {renderTag}
          </Select.MultipleTrigger>
          <Select.Dialog>
            <WindowSearch
              onSearch={(next) => {
                setTerm({ term: next, typedAt: TimeStamp.now() });
                search(next);
              }}
              typed={typed}
              onEnter={take}
            />
            <LiveRow span={span} onChange={setRolling} />
            <Select.List bordered borderColor={6} grow rounded full="x">
              {typed != null && (
                <WindowOption itemKey={typed.key} term={term} window={typed.window} />
              )}
              <Select.Items<string> emptyContent={empty}>{rangeItem}</Select.Items>
            </Select.List>
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>
    </Input.Item>
  );
};
