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
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { type position } from "@synnaxlabs/lyra/position";
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
  type TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { Range } from "@/platform/range";
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

const FavoriteItem = (props: List.ItemProps<string>): ReactElement | null => {
  const range = Range.useResolve(props.itemKey);
  const { data: parent } = Ranger.useResultParent({ key: props.itemKey });
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
};

const favoriteItem = Component.renderProp(FavoriteItem);

interface RollingTagProps {
  itemKey: string;
  window: DynamicWindow;
}

const RollingTag = ({ itemKey, window }: RollingTagProps): ReactElement => {
  const { onSelect } = Select.useItemState(itemKey);
  return (
    <Tag.Tag icon={liveIcon} onClose={onSelect} level="small" size="small">
      {new TimeSpan(window.timeSpan).toString()}
    </Tag.Tag>
  );
};

interface KeyTagProps {
  range: ranger.KeyRef;
}

const KeyTag = ({ range: { key } }: KeyTagProps): ReactElement => {
  const { onSelect } = Select.useItemState(key);
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
  itemKey: string;
  window: StaticWindow;
  name?: string;
}

const StaticTag = ({ itemKey, window, name }: StaticTagProps): ReactElement => {
  const { onSelect } = Select.useItemState(itemKey);
  const timeRange = resolveWindow(window, TimeStamp.now());
  // The day prefix changes at midnight.
  const now = useNow((now) => Telem.Text.nextMidnight(now));
  const exact = formatExact(timeRange);
  return (
    <Tag.Tag
      icon={<Ranger.StageIcon timeRange={timeRange} />}
      onClose={onSelect}
      level="small"
      size="small"
      tooltip={name == null ? exact : `${name}: ${exact}`}
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

const parseSpan = (term: string): TimeSpan | undefined => {
  const span = TimeSpan.parse(term);
  return span == null || span.isZero ? undefined : span;
};

interface DurationSearchProps {
  typed?: TimeSpan;
  onSearch: (term: string) => void;
  onDuration: (span: TimeSpan) => void;
}

// Enter on a typed duration sets the rolling window and closes the dialog. The wrapper
// packs, so the field takes the dialog's top corners.
const DurationSearch = ({
  typed,
  onSearch,
  onDuration,
}: DurationSearchProps): ReactElement => {
  const { close } = Dialog.useContext();
  return (
    <Flex.Box
      y
      pack
      onKeyDown={(e) => {
        if (e.key !== "Enter" || typed == null) return;
        // Keeps the list's window-level Enter trigger from clicking a favorite.
        e.stopPropagation();
        onDuration(typed);
        close();
      }}
    >
      <Select.Search
        placeholder="Search pinned ranges or type a duration"
        onSearch={onSearch}
      />
    </Flex.Box>
  );
};

const ABOVE: position.LocationPreference = {
  targetCorner: "top",
  dialogCorner: "bottom",
};

const emptyContent = (
  <Status.Summary center variant="disabled">
    No favorite ranges found
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
 * Selects what the plot shows: at most one rolling window, favorite ranges, and static
 * windows. A duration typed into the search lists a live option, and Enter sets it. A
 * static window's tag edits the window in place.
 */
export const RangeSelect = (props: RangeSelectProps): ReactElement => {
  const refs = LinePlot.useRanges();
  const states = Session.Range.useSelectMultiple();
  const dispatch = LinePlot.useSingleDispatch();
  const favorites = Range.useResolveMultiple().filter((r) => r.variant !== "dynamic");
  const { data, search } = List.useStaticData<string>({ data: favorites });
  const [term, setTerm] = useState("");
  const entryOf = (ref: ranger.Ref): Entry => ({
    ref,
    key:
      ref.variant === "key"
        ? ref.key
        : (Session.Range.findKey(ref, states) ?? windowKey(ref)),
  });
  const entries = refs.map(entryOf);
  const typedSpan = parseSpan(term);
  const typed =
    typedSpan == null
      ? undefined
      : { span: typedSpan, key: entryOf(rollingRef(typedSpan)).key };
  const rolling = refs.find(isRolling);
  const span = rolling == null ? undefined : new TimeSpan(rolling.window.timeSpan);
  const setRanges = (ranges: ranger.Ref[]) =>
    dispatch(lineplot.setRanges({ ranges: dedupe(ranges) }));
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
  const handleChange = (keys: string[]) => {
    if (
      typed != null &&
      keys.includes(typed.key) &&
      !entries.some(({ key }) => key === typed.key)
    )
      return setRolling(typed.span);
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
  const renderTag = ({ itemKey }: Select.MultipleTagProps<string>) => {
    const entry = entries.find(({ key }) => key === itemKey);
    if (entry == null) return null;
    const { ref } = entry;
    if (ref.variant === "key") return <KeyTag key={itemKey} range={ref} />;
    if (ref.window.variant === "dynamic")
      return <RollingTag key={itemKey} itemKey={itemKey} window={ref.window} />;
    const state = states.find((s) => s.key === itemKey);
    return (
      <StaticTag
        key={itemKey}
        itemKey={itemKey}
        window={ref.window}
        name={state?.variant === "static" ? state.name : undefined}
      />
    );
  };
  return (
    <Input.Item x label="Ranges" {...props}>
      <Dialog.Frame variant="floating" location={ABOVE}>
        <Select.Frame<string>
          multiple
          value={entries.map(({ key }) => key)}
          onChange={handleChange}
          data={data}
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
            <DurationSearch
              typed={typed?.span}
              onSearch={(next) => {
                setTerm(next);
                search(next);
              }}
              onDuration={setRolling}
            />
            <LiveRow span={span} onChange={setRolling} />
            <Select.List bordered borderColor={6} grow rounded full="x">
              {typed != null && (
                // Its text is the term, so the search that typed it never hides it.
                <Select.Item itemKey={typed.key} textValue={term} justify="between">
                  <Text.Text>
                    {liveIcon}
                    {typed.span.toString()}
                  </Text.Text>
                  <Text.Text level="small" color={9}>
                    Live window
                  </Text.Text>
                </Select.Item>
              )}
              <Select.Items<string> emptyContent={typed == null ? emptyContent : null}>
                {favoriteItem}
              </Select.Items>
            </Select.List>
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>
    </Input.Item>
  );
};
