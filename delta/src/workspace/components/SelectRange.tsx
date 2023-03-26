// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  InputItemProps,
  Select,
  SelectMultipleProps,
  SelectProps,
  Input,
  Space,
  Text,
  Status,
  Button,
} from "@synnaxlabs/pluto";

import { Range } from "../store";

import { rangeWindowLayout } from "./DefineRange";
import { rangeListColumns } from "./RangesList";

import { useLayoutPlacer } from "@/layout";

export interface SelectMultipleRangesProps
  extends Omit<SelectMultipleProps<Range>, "columns"> {}

export const SelectMultipleRanges = (props: SelectMultipleRangesProps): JSX.Element => (
  <Select.Multiple columns={rangeListColumns} tagKey="name" {...props} />
);

export interface SelectRangeProps extends Omit<SelectProps<Range>, "columns"> {}

export const SelectRange = (props: SelectRangeProps): JSX.Element => (
  <Select columns={rangeListColumns} {...props} tagKey="name" />
);

export interface SelectMultipleRangesInputItemProps
  extends Omit<
    InputItemProps<readonly string[], readonly string[], SelectMultipleRangesProps>,
    "label"
  > {}

const SelectRangesEmptyContent = (): JSX.Element => {
  const newLayout = useLayoutPlacer();
  return (
    <Space.Centered style={{ height: 150 }} direction="x">
      <Status.Text variant="disabled" hideIcon>
        No Ranges:
      </Status.Text>
      <Button
        variant="outlined"
        onClick={() => {
          newLayout({
            ...rangeWindowLayout,
            key: rangeWindowLayout.key,
          });
        }}
      >
        Define a Range
      </Button>
    </Space.Centered>
  );
};

export const SelectMultipleRangesInputItem = (
  props: SelectMultipleRangesInputItemProps
): JSX.Element => (
  <Input.Item<readonly string[], readonly string[], SelectMultipleRangesProps>
    direction="x"
    label="Ranges:"
    emptyContent={<SelectRangesEmptyContent />}
    {...props}
  >
    {SelectMultipleRanges}
  </Input.Item>
);

export interface SelectRangeInputItemProps
  extends Omit<InputItemProps<string, string, SelectRangeProps>, "label"> {}

export const SelectRangeInputItem = (props: SelectRangeInputItemProps): JSX.Element => (
  <Input.Item<string, string, SelectRangeProps> label="Range:" {...props}>
    {SelectRange}
  </Input.Item>
);
