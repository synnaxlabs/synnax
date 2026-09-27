// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/range/Select.css";

import { ranger } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Ranger, TimeSpan } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import { CSS } from "@/platform/css";
import {
  type Resolved,
  useResolve,
  useResolveMultiple,
} from "@/platform/range/resolve";
import { type Session } from "@/session";

const dynamicIcon = <Icon.Dynamic className={CSS.BE("range-select", "dynamic-icon")} />;

const StaticListItem = Component.renderProp(
  (
    props: List.ItemProps<string> & {
      range: Exclude<Resolved, Session.Range.DynamicState>;
    },
  ) => {
    const { range } = props;
    const { data: parent } = Ranger.useResultParent({
      id: ranger.ontologyID(range.key),
    });
    return (
      <Select.Item {...props} justify="between">
        <Ranger.Breadcrumb
          key={range.key}
          name={range.name}
          parent={parent}
          timeRange={range.timeRange}
        />
        <Telem.Text.TimeRange level="small">{range.timeRange}</Telem.Text.TimeRange>
      </Select.Item>
    );
  },
);

/** The key that selects the consumer's custom window. */
export const CUSTOM_KEY = "custom";

// A selection reports only keys, so the option's key carries the window it sets.
const DRAFT_PREFIX = `${CUSTOM_KEY}:`;

const CustomOption = (): ReactElement | null => {
  const term = Select.useSearchTerm();
  const span = TimeSpan.parse(term);
  if (span == null || span.isZero) return null;
  // The search hides fixed items whose text lacks the term, so the option echoes it.
  return (
    <Select.Item itemKey={`${DRAFT_PREFIX}${span.valueOf()}`} justify="between">
      <Text.Text>
        <Icon.Add />
        Custom
      </Text.Text>
      <Text.Text>
        {term.trim()}
        {dynamicIcon}
      </Text.Text>
    </Select.Item>
  );
};

const customOption = <CustomOption />;

const listItem = Component.renderProp((props: List.ItemProps<string>) => {
  const range = useResolve(props.itemKey);
  if (range == null || range.variant === "dynamic") return null;
  return <StaticListItem {...props} range={range} />;
});

interface RenderTagProps {
  itemKey: string;
}

const RangeTag = ({ itemKey }: RenderTagProps): ReactElement | null => {
  const range = useResolve(itemKey);
  const { onSelect } = Select.useItemState(itemKey);
  return (
    <Tag.Tag
      icon={range?.variant === "dynamic" ? dynamicIcon : <Icon.Range />}
      onClose={onSelect}
      level="small"
      size="small"
    >
      {range?.name ?? itemKey}
    </Tag.Tag>
  );
};

interface CustomProps {
  /** The rolling window, in nanoseconds, that the custom key resolves to. */
  customSpan?: number;
  /** Sets the custom window and selects the custom key. */
  onCustomChange: (span: number) => void;
}

interface CustomTagProps extends Pick<CustomProps, "onCustomChange"> {
  span: number;
}

const CustomTag = ({ span, onCustomChange }: CustomTagProps): ReactElement => {
  const { onSelect } = Select.useItemState(CUSTOM_KEY);
  return (
    <Tag.Tag
      icon={dynamicIcon}
      onClose={onSelect}
      level="small"
      size="small"
      className={CSS.BE("range-select", "custom")}
    >
      <Input.TimeSpan
        value={span}
        onChange={onCustomChange}
        variant="text"
        size="small"
      />
    </Tag.Tag>
  );
};

const isLive = (range: Resolved): range is Session.Range.DynamicState =>
  range.variant === "dynamic";

interface LiveRowProps {
  ranges: Session.Range.DynamicState[];
  value: string[];
  onChange: (value: string[]) => void;
}

// A frame of its own, so its toggles stay out of the list's arrow keys and search.
const LiveRow = ({ ranges, value, onChange }: LiveRowProps): ReactElement => (
  <Select.Buttons
    multiple
    allowNone
    value={value}
    onChange={onChange}
    enableTriggers={false}
    className={CSS.BE("range-select", "live")}
    wrap
  >
    {ranges.map(({ key, span }) => (
      <Select.Item key={key} itemKey={key}>
        {new TimeSpan(span).toString()}
      </Select.Item>
    ))}
  </Select.Buttons>
);

const emptyContent = (
  <Status.Summary center variant="disabled">
    No favorite ranges found
  </Status.Summary>
);

interface SelectMultipleRangesProps extends CustomProps {
  value: string[];
  onChange: (value: string[]) => void;
}

const SelectMultipleRanges = ({
  value,
  onChange,
  customSpan = 0,
  onCustomChange,
}: SelectMultipleRangesProps): ReactElement => {
  const entries = useResolveMultiple();
  const live = useMemo(() => entries.filter(isLive), [entries]);
  const favorites = useMemo(() => entries.filter((r) => !isLive(r)), [entries]);
  const liveKeys = useMemo(() => new Set(live.map(({ key }) => key)), [live]);
  const { data, retrieve } = List.useStaticData<string>({ data: favorites });
  const { fetchMore, search } = List.usePager({ retrieve });
  const handleChange = useCallback(
    (keys: string[]) => {
      const draft = keys.find((k) => k.startsWith(DRAFT_PREFIX));
      if (draft == null) return onChange(keys);
      onCustomChange(Number(draft.slice(DRAFT_PREFIX.length)));
    },
    [onChange, onCustomChange],
  );
  const selectedLive = useMemo(
    () => value.filter((k) => liveKeys.has(k)),
    [value, liveKeys],
  );
  const handleLiveChange = useCallback(
    (keys: string[]) =>
      onChange([
        ...value.filter((k) => !liveKeys.has(k) || keys.includes(k)),
        ...keys.filter((k) => !value.includes(k)),
      ]),
    [onChange, value, liveKeys],
  );
  const renderTag = useCallback(
    ({ itemKey }: Select.MultipleTagProps<string>) =>
      itemKey === CUSTOM_KEY ? (
        <CustomTag key={itemKey} span={customSpan} onCustomChange={onCustomChange} />
      ) : (
        <RangeTag key={itemKey} itemKey={itemKey} />
      ),
    [customSpan, onCustomChange],
  );
  return (
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
          <LiveRow ranges={live} value={selectedLive} onChange={handleLiveChange} />
          <Select.List bordered borderColor={6} grow rounded full="x">
            {customOption}
            <Select.Items<string> emptyContent={emptyContent}>{listItem}</Select.Items>
          </Select.List>
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};

export interface SelectMultipleInputItemProps
  extends
    Omit<Input.ItemProps, "label" | "onChange" | "children">,
    SelectMultipleRangesProps {}

export const SelectMultipleInputItem = ({
  value,
  onChange,
  customSpan,
  onCustomChange,
  ...rest
}: SelectMultipleInputItemProps): ReactElement => (
  <Input.Item x label="Ranges" {...rest}>
    <SelectMultipleRanges
      value={value}
      onChange={onChange}
      customSpan={customSpan}
      onCustomChange={onCustomChange}
    />
  </Input.Item>
);
