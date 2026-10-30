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
import {
  type DynamicWindow,
  dynamicWindow,
  type StaticWindow,
  staticWindow,
  TimeRange,
  TimeSpan,
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
  itemKey: string;
  window: StaticWindow;
  name?: string;
  onChange: (timeRange: TimeRange) => void;
}

const StaticTag = ({
  itemKey,
  window,
  name,
  onChange,
}: StaticTagProps): ReactElement => {
  const { onSelect } = Select.useItemState(itemKey);
  const start = Number(window.timeRange.start);
  const end = Number(window.timeRange.end);
  const handleChange = (next: TimeRange) => {
    if (next.isValid && !next.span.isZero) onChange(next);
  };
  return (
    <Tag.Tag
      icon={<Icon.Range />}
      onClose={onSelect}
      level="small"
      size="small"
      tooltip={name}
    >
      <Input.DateTime
        value={start}
        onChange={(next) => handleChange(new TimeRange(next, end))}
        aria-label="Start"
        variant="text"
        size="small"
        bound="start"
        anchors={{ end }}
      />
      <Icon.Arrow.Right />
      <Input.DateTime
        value={end}
        onChange={(next) => handleChange(new TimeRange(start, next))}
        aria-label="End"
        variant="text"
        size="small"
        bound="end"
        anchors={{ start }}
        sharedDay={start}
      />
    </Tag.Tag>
  );
};

interface LiveRowProps {
  span?: TimeSpan;
  onChange: (span?: TimeSpan) => void;
}

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
      value={span == null ? undefined : Number(span.valueOf())}
      onChange={(key: number | null) =>
        onChange(key == null ? undefined : new TimeSpan(key))
      }
      wrap
    >
      {ROLLING_PRESETS.map((preset) => (
        <Select.Item key={preset} itemKey={preset}>
          {new TimeSpan(preset).toString()}
        </Select.Item>
      ))}
    </Select.Buttons>
  </Flex.Box>
);

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
 * windows. A duration typed into the search sets the rolling window on Enter, and a
 * static window's tag edits the window in place.
 */
export const RangeSelect = (props: RangeSelectProps): ReactElement => {
  const refs = LinePlot.useRanges();
  const states = Session.Range.useSelectMultiple();
  const dispatch = LinePlot.useSingleDispatch();
  const favorites = Range.useResolveMultiple().filter((r) => r.variant !== "dynamic");
  const { data, retrieve } = List.useStaticData<string>({ data: favorites });
  const { fetchMore, search } = List.usePager({ retrieve });
  const [term, setTerm] = useState("");
  const entries = refs.map((ref): Entry => ({
    ref,
    key:
      ref.variant === "key"
        ? ref.key
        : (Session.Range.findKey(ref, states) ?? windowKey(ref)),
  }));
  const rolling = refs.find(isRolling);
  const span = rolling == null ? undefined : new TimeSpan(rolling.window.timeSpan);
  const setRanges = (ranges: ranger.Ref[]) =>
    dispatch(lineplot.setRanges({ ranges: dedupe(ranges) }));
  const replace = (prev: ranger.Ref, next: ranger.Ref) =>
    setRanges(refs.map((r) => (r === prev ? next : r)));
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
        onChange={(timeRange) =>
          replace(ref, { variant: "window", window: staticWindow(timeRange) })
        }
      />
    );
  };
  return (
    <Input.Item x label="Ranges" {...props}>
      <Dialog.Frame variant="floating">
        <Select.Frame<string>
          multiple
          value={entries.map(({ key }) => key)}
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
            <Flex.Box
              y
              empty
              onKeyDown={(e) => {
                if (e.key !== "Enter") return;
                const typed = TimeSpan.parse(term);
                if (typed != null && !typed.isZero) setRolling(typed);
              }}
            >
              <Select.Search
                placeholder="Search favorites or type a duration"
                onSearch={(next) => {
                  setTerm(next);
                  search(next);
                }}
              />
            </Flex.Box>
            <LiveRow span={span} onChange={setRolling} />
            <Select.List bordered borderColor={6} grow rounded full="x">
              <Select.Items<string> emptyContent={emptyContent}>
                {favoriteItem}
              </Select.Items>
            </Select.List>
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>
    </Input.Item>
  );
};
