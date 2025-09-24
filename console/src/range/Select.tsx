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
  Icon,
  Input,
  List,
  Select,
  Tag,
  Telem,
  Text,
  TimeSpan,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useSelect, useSelectKeys, useSelectMultiple } from "@/range/selectors";
import { type Range } from "@/range/slice";

const dynamicIcon = (
  <Icon.Dynamic style={{ color: "var(--pluto-error-p1)", filter: "opacity(0.8)" }} />
);

const listItem = Component.renderProp((props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const range = useSelect(itemKey);
  if (range == null) return null;
  const { variant, name } = range;
  return (
    <Select.ListItem {...props}>
      <Text.Text style={{ width: 100 }}>{name}</Text.Text>
      {variant === "dynamic" ? (
        <Text.Text>
          {dynamicIcon}
          {new TimeSpan(range.span).toString()}
        </Text.Text>
      ) : (
        <Telem.Text.TimeRange level="small">{range.timeRange}</Telem.Text.TimeRange>
      )}
    </Select.ListItem>
  );
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
      status={range == null ? "error" : undefined}
    >
      {range?.name ?? itemKey}
    </Tag.Tag>
  );
};

export const renderTag = Component.renderProp(RangeTag);

export interface SelectMultipleRangesProps extends Input.Control<string[]> {}

const SelectMultipleRanges = ({
  value,
  onChange,
}: SelectMultipleRangesProps): ReactElement => {
  const entries = useSelectMultiple();
  const { data, retrieve } = List.useStaticData<string>({ data: entries });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Select.Multiple<string, Range>
      resourceName="Range"
      data={data}
      icon={<Icon.Range />}
      renderTag={renderTag}
      onFetchMore={fetchMore}
      onSearch={search}
      value={value}
      onChange={onChange}
    >
      {listItem}
    </Select.Multiple>
  );
};

interface SelectSingleRangeProps extends Input.Control<string> {}

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
