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
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Ranger, TimeSpan } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/platform/css";
import {
  type Resolved,
  useResolve,
  useResolveMultiple,
} from "@/platform/range/resolve";
import { type Session } from "@/session";

interface SelectMultipleRangesProps extends Omit<
  Select.MultipleProps<string, Session.Range.State>,
  "resourceName" | "data" | "children"
> {}

const dynamicIcon = <Icon.Dynamic className={CSS.BE("range-select", "dynamic-icon")} />;

const DynamicListItem = Component.renderProp(
  (props: List.ItemProps<string> & { range: Session.Range.DynamicState }) => {
    const { range } = props;
    return (
      <Select.Item {...props} justify="between">
        <Text.Text className={CSS.BE("range-select", "dynamic-name")}>
          {range.name}
        </Text.Text>
        <Text.Text>
          {new TimeSpan(range.span).toString()}
          {dynamicIcon}
        </Text.Text>
      </Select.Item>
    );
  },
);

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
  if (range == null) return null;
  if (range.variant === "dynamic") return <DynamicListItem {...props} range={range} />;
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
    <Tag.Tag icon={dynamicIcon} onClose={onSelect} level="small" size="small">
      <Input.TimeSpan
        value={span}
        onChange={onCustomChange}
        variant="text"
        size="small"
      />
    </Tag.Tag>
  );
};

const SelectMultipleRanges = ({
  onChange,
  customSpan = 0,
  onCustomChange,
  ...rest
}: SelectMultipleRangesProps & CustomProps): ReactElement => {
  const entries = useResolveMultiple();
  const { data, retrieve } = List.useStaticData<string>({ data: entries });
  const { fetchMore, search } = List.usePager({ retrieve });
  const handleChange = useCallback(
    (keys: string[], extra: Select.UseOnChangeExtra<string>) => {
      const draft = keys.find((k) => k.startsWith(DRAFT_PREFIX));
      if (draft == null) return onChange(keys, extra);
      onCustomChange(Number(draft.slice(DRAFT_PREFIX.length)));
    },
    [onChange, onCustomChange],
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
    <Select.Multiple<string, Session.Range.State>
      icon={<Icon.Range />}
      renderTag={renderTag}
      onFetchMore={fetchMore}
      onSearch={search}
      {...rest}
      onChange={handleChange}
      resourceName="range"
      data={data}
      fixedItems={customOption}
    >
      {listItem}
    </Select.Multiple>
  );
};

export interface SelectMultipleInputItemProps
  extends
    Omit<Input.ItemProps, "label" | "onChange" | "children">,
    Omit<SelectMultipleRangesProps, "status">,
    CustomProps {
  value: string[];
  onChange: (value: string[]) => void;
  selectProps?: Partial<SelectMultipleRangesProps>;
}

export const SelectMultipleInputItem = ({
  value,
  onChange,
  customSpan,
  onCustomChange,
  selectProps,
  ...rest
}: SelectMultipleInputItemProps): ReactElement => (
  <Input.Item x label="Ranges" {...rest}>
    <SelectMultipleRanges
      value={value}
      onChange={onChange}
      customSpan={customSpan}
      onCustomChange={onCustomChange}
      {...selectProps}
    />
  </Input.Item>
);
