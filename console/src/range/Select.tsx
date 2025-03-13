// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  Input,
  type List,
  Ranger,
  Select,
  Status,
  Tag,
  Text,
  TimeSpan,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/range/Create";
import { type Range } from "@/range/slice";

interface SelectMultipleRangesProps extends Select.MultipleProps<string, Range> {}

const dynamicIcon = (
  <Icon.Dynamic style={{ color: "var(--pluto-error-p1)", filter: "opacity(0.8)" }} />
);

const listColumns: Array<List.ColumnSpec<string, Range>> = [
  { key: "name", name: "Name", weight: 450 },
  {
    key: "start",
    name: "Start",
    width: 150,
    render: ({ entry }) => {
      if (entry.variant === "dynamic")
        return (
          <Text.WithIcon level="p" startIcon={dynamicIcon} shade={7}>
            {new TimeSpan(entry.span).toString()}
          </Text.WithIcon>
        );
      return <Ranger.TimeRangeChip level="small" timeRange={entry.timeRange} />;
    },
  },
];

const RenderTag = ({
  entry,
  onClose,
}: Select.MultipleTagProps<string, Range>): ReactElement => (
  <Tag.Tag
    icon={entry?.variant === "dynamic" ? dynamicIcon : <Icon.Range />}
    onClose={onClose}
    shade={9}
    level="small"
  >
    {entry?.name}
  </Tag.Tag>
);

const renderTag = componentRenderProp(RenderTag);

const SelectMultipleRanges = (props: SelectMultipleRangesProps): ReactElement => (
  <Select.Multiple
    columns={listColumns}
    entryRenderKey="name"
    renderTag={renderTag}
    {...props}
  />
);

interface SelectSingleRangeProps extends Select.SingleProps<string, Range> {}

const SelectRange = (props: SelectSingleRangeProps): ReactElement => (
  <Select.Single columns={listColumns} {...props} entryRenderKey="name" />
);

interface SelectMultipleInputItemProps
  extends Omit<Input.ItemProps, "label" | "onChange">,
    Pick<SelectMultipleRangesProps, "data"> {
  value: string[];
  onChange: (value: string[]) => void;
  selectProps?: Partial<SelectMultipleRangesProps>;
}

const SelectEmptyContent = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <Align.Center style={{ height: 150 }} direction="x">
      <Status.Text variant="disabled" hideIcon>
        No Ranges:
      </Status.Text>
      <Button.Button variant="outlined" onClick={() => placeLayout(CREATE_LAYOUT)}>
        Define a Range
      </Button.Button>
    </Align.Center>
  );
};

export const SelectMultipleInputItem = ({
  value,
  onChange,
  data,
  selectProps,
  ...rest
}: SelectMultipleInputItemProps): ReactElement => (
  <Input.Item direction="x" label="Ranges" {...rest}>
    <SelectMultipleRanges
      data={data}
      value={value}
      onChange={onChange}
      emptyContent={<SelectEmptyContent />}
      {...selectProps}
    />
  </Input.Item>
);

interface SelectInputItemProps
  extends Omit<Input.ItemProps, "label" | "onChange">,
    Input.Control<string>,
    Pick<SelectSingleRangeProps, "data"> {
  selectProps?: Partial<SelectSingleRangeProps>;
}

export const SelectInputItem = ({
  value,
  onChange,
  data,
  selectProps,
  ...rest
}: SelectInputItemProps): ReactElement => (
  <Input.Item label="Range:" {...rest}>
    <SelectRange
      value={value}
      onChange={onChange}
      data={data}
      emptyContent={<SelectEmptyContent />}
      {...selectProps}
    />
  </Input.Item>
);
