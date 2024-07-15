// Copyright 2024 Synnax Labs, Inc.
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
  Input,
  List,
  Ranger,
  Select,
  Status,
  Text,
  TimeSpan,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Layout } from "@/layout";
import { createEditLayout } from "@/range/EditLayout";
import { type Range } from "@/range/slice";

export interface SelectMultipleRangesProps
  extends Select.MultipleProps<string, Range> {}

export const listColumns: Array<List.ColumnSpec<string, Range>> = [
  {
    key: "name",
    name: "Name",
    weight: 450,
  },
  {
    key: "start",
    name: "Start",
    width: 150,
    render: ({ entry }) => {
      if (entry.variant === "dynamic")
        return (
          <Text.WithIcon
            level="p"
            startIcon={
              <Icon.Dynamic
                style={{ color: "var(--pluto-error-p1)", filter: "opacity(0.8)" }}
              />
            }
            shade={7}
          >
            {new TimeSpan(entry.span).toString()}
          </Text.WithIcon>
        );
      return <Ranger.TimeRangeChip timeRange={entry.timeRange} />;
    },
  },
];

export const SelectMultipleRanges = (
  props: SelectMultipleRangesProps,
): ReactElement => (
  <Select.Multiple columns={listColumns} entryRenderKey="name" {...props} />
);

export interface SelectSingleRangeProps extends Select.SingleProps<string, Range> {}

export const SelectRange = (props: SelectSingleRangeProps): ReactElement => (
  <Select.Single columns={listColumns} {...props} entryRenderKey="name" />
);

export interface SelectMultipleInputItemProps
  extends Omit<Input.ItemProps, "label" | "onChange">,
    Pick<SelectMultipleRangesProps, "data"> {
  value: string[];
  onChange: (value: string[]) => void;
}

const SelectEmptyContent = (): ReactElement => {
  const newLayout = Layout.usePlacer();
  return (
    <Align.Center style={{ height: 150 }} direction="x">
      <Status.Text variant="disabled" hideIcon>
        No Ranges:
      </Status.Text>
      <Button.Button
        variant="outlined"
        onClick={() => {
          newLayout(createEditLayout());
        }}
      >
        Define a Range
      </Button.Button>
    </Align.Center>
  );
};

export const SelectMultipleInputItem = ({
  value,
  onChange,
  data,
  ...props
}: SelectMultipleInputItemProps): ReactElement => (
  <Input.Item direction="x" label="Ranges" {...props}>
    <SelectMultipleRanges
      data={data}
      value={value}
      onChange={onChange}
      emptyContent={<SelectEmptyContent />}
    />
  </Input.Item>
);

export interface SelectInputItemProps
  extends Omit<Input.ItemProps, "label" | "onChange">,
    Input.Control<string>,
    Pick<SelectSingleRangeProps, "data"> {}

export const SelectInputItem = ({
  value,
  onChange,
  data,
  ...props
}: SelectInputItemProps): ReactElement => (
  <Input.Item label="Range:" {...props}>
    <SelectRange
      value={value}
      onChange={onChange}
      data={data}
      emptyContent={<SelectEmptyContent />}
    />
  </Input.Item>
);
