// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import {
  Component,
  Icon,
  Input,
  List,
  Ranger,
  Select,
  Tag,
  Telem,
  Text,
  TimeSpan,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useSelect, useSelectMultiple } from "@/range/selectors";
import { type Range } from "@/range/slice";
import { type DynamicRange, type StaticRange } from "@/range/types";

interface SelectMultipleRangesProps extends Omit<
  Select.MultipleProps<string, Range>,
  "resourceName" | "data" | "children"
> {}

const dynamicIcon = (
  <Icon.Dynamic style={{ color: "var(--pluto-error-p1)", filter: "opacity(0.8)" }} />
);

const DynamicListItem = Component.renderProp(
  (props: List.ItemProps<string> & { range: DynamicRange }) => {
    const { range } = props;
    return (
      <Select.ListItem {...props} justify="between">
        <Text.Text style={{ width: 100 }}>{range.name}</Text.Text>
        <Text.Text>
          {new TimeSpan(range.span).toString()}
          {dynamicIcon}
        </Text.Text>
      </Select.ListItem>
    );
  },
);

const StaticListItem = Component.renderProp(
  (props: List.ItemProps<string> & { range: StaticRange }) => {
    const { range } = props;
    const parent = Ranger.useRetrieveParent({ id: ranger.ontologyID(range.key) }).data;
    return (
      <Select.ListItem {...props} justify="between">
        <Ranger.Breadcrumb
          key={range.key}
          name={range.name}
          parent={parent}
          timeRange={range.timeRange}
        />
        <Telem.Text.TimeRange level="small">{range.timeRange}</Telem.Text.TimeRange>
      </Select.ListItem>
    );
  },
);

const listItem = Component.renderProp((props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const range = useSelect(itemKey);
  if (range == null) return null;
  const { variant } = range;
  if (variant === "dynamic") return <DynamicListItem {...props} range={range} />;
  return <StaticListItem {...props} range={range} />;
});

interface RenderTagProps {
  itemKey: string;
}

const RangeTag = ({ itemKey }: RenderTagProps): ReactElement | null => {
  const range = useSelect(itemKey);
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

const renderTag = Component.renderProp(RangeTag);

const SelectMultipleRanges = (props: SelectMultipleRangesProps): ReactElement => {
  const entries = useSelectMultiple();
  const { data, retrieve } = List.useStaticData<string>({ data: entries });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Multiple<string, Range>
      icon={<Icon.Range />}
      renderTag={renderTag}
      onFetchMore={fetchMore}
      onSearch={search}
      {...props}
      resourceName="range"
      data={data}
    >
      {listItem}
    </Select.Multiple>
  );
};

export interface SelectMultipleInputItemProps
  extends
    Omit<Input.ItemProps, "label" | "onChange" | "children">,
    Omit<SelectMultipleRangesProps, "status"> {
  value: string[];
  onChange: (value: string[]) => void;
  selectProps?: Partial<SelectMultipleRangesProps>;
}

export const SelectMultipleInputItem = ({
  value,
  onChange,
  selectProps,
  ...rest
}: SelectMultipleInputItemProps): ReactElement => (
  <Input.Item x label="Ranges" {...rest}>
    <SelectMultipleRanges value={value} onChange={onChange} {...selectProps} />
  </Input.Item>
);
