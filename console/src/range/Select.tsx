// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Component,
  Flux,
  Icon,
  Input,
  List,
  Ranger,
  Select,
  Tag,
  Text,
  TimeSpan,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useSelect, useSelectKeys, useSelectMultiple } from "@/range/selectors";
import { type Range } from "@/range/slice";

interface SelectMultipleRangesProps
  extends Omit<
    Select.MultipleProps<string, Range>,
    "resourceName" | "data" | "children"
  > {}

const dynamicIcon = (
  <Icon.Dynamic style={{ color: "var(--pluto-error-p1)", filter: "opacity(0.8)" }} />
);

const listItem = Component.renderProp((props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const range = useSelect(itemKey);
  const { selected, onSelect } = Select.useItemState(itemKey);
  if (range == null) return null;
  const { variant, name } = range;
  return (
    <List.Item {...props} selected={selected} onSelect={onSelect}>
      <Text.Text level="p" style={{ width: 100 }}>
        {name}
      </Text.Text>
      {variant === "dynamic" ? (
        <Text.WithIcon level="p" shade={11} startIcon={dynamicIcon}>
          {new TimeSpan(range.span).toString()}
        </Text.WithIcon>
      ) : (
        <Ranger.TimeRangeChip level="small" timeRange={range.timeRange} />
      )}
    </List.Item>
  );
});

interface RenderTagProps {
  itemKey: string;
}

const RangeTag = ({ itemKey }: RenderTagProps): ReactElement | null => {
  const range = useSelect(itemKey);
  const { onSelect } = Select.useItemState(itemKey);
  if (range == null) return null;
  return (
    <Tag.Tag
      icon={range?.variant === "dynamic" ? dynamicIcon : <Icon.Range />}
      onClose={onSelect}
      shade={11}
      level="small"
      size="small"
    >
      {range.name}
    </Tag.Tag>
  );
};

export const renderTag = Component.renderProp(RangeTag);

const SelectMultipleRanges = (props: SelectMultipleRangesProps): ReactElement => {
  const entries = useSelectMultiple();
  const { data, retrieve } = List.useStaticData<string>({ data: entries });
  const { onFetchMore, onSearch } = Flux.usePager({ retrieve });
  return (
    <Select.Multiple<string, Range>
      resourceName="Range"
      data={data}
      icon={<Icon.Range />}
      renderTag={renderTag}
      onFetchMore={onFetchMore}
      onSearch={onSearch}
      {...props}
    >
      {listItem}
    </Select.Multiple>
  );
};

interface SelectSingleRangeProps
  extends Omit<
    Select.SingleProps<string, Range>,
    "resourceName" | "data" | "children"
  > {}

const SelectRange = ({ value, onChange }: SelectSingleRangeProps): ReactElement => {
  const data = useSelectKeys();
  return (
    <Select.Single<string, Range>
      resourceName="Range"
      value={value}
      onChange={onChange}
      data={data}
      icon={<Icon.Range />}
    >
      {listItem}
    </Select.Single>
  );
};

interface SelectMultipleInputItemProps
  extends Omit<Input.ItemProps, "label" | "onChange" | "children">,
    SelectMultipleRangesProps {
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

interface SelectInputItemProps
  extends Omit<Input.ItemProps, "label" | "onChange" | "children">,
    SelectSingleRangeProps {
  selectProps?: Partial<SelectSingleRangeProps>;
}

export const SelectInputItem = ({
  value,
  onChange,
  selectProps,
  ...rest
}: SelectInputItemProps): ReactElement => (
  <Input.Item label="Range:" {...rest}>
    <SelectRange value={value} onChange={onChange} {...selectProps} />
  </Input.Item>
);
