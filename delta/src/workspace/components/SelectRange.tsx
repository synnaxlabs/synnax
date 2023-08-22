// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Input, Status, Button } from "@synnaxlabs/pluto";

import { useLayoutPlacer } from "@/layout";

import { Range } from "../store";

import { rangeWindowLayout } from "./DefineRange";
import { rangeListColumns } from "./RangesList";

export interface SelectMultipleRangesProps
  extends Omit<SelectMultipleProps<string, Range>, "columns"> {}

export const SelectMultipleRanges = (
  props: SelectMultipleRangesProps
): ReactElement => (
  <Select.Multiple columns={rangeListColumns} tagKey="name" {...props} />
);

export interface SelectRangeProps extends Omit<SelectProps<string, Range>, "columns"> {}

export const SelectRange = (props: SelectRangeProps): ReactElement => (
  <Select columns={rangeListColumns} {...props} tagKey="name" />
);

export interface SelectMultipleRangesInputItemProps
  extends Omit<
    InputItemProps<readonly string[], readonly string[], SelectMultipleRangesProps>,
    "label"
  > {}

const SelectRangesEmptyContent = (): ReactElement => {
  const newLayout = useLayoutPlacer();
  return (
    <Space.Centered style={{ height: 150 }} direction="x">
      <Status.Text variant="disabled" hideIcon>
        No Ranges:
      </Status.Text>
      <Button
        variant="outlined"
        onClick={() => {
          newLayout(rangeWindowLayout);
        }}
      >
        Define a Range
      </Button>
    </Space.Centered>
  );
};

export const SelectMultipleRangesInputItem = (
  props: SelectMultipleRangesInputItemProps
): ReactElement => (
  <Input.Item<readonly string[], readonly string[], SelectMultipleRangesProps>
    direction="x"
    label="Ranges"
    emptyContent={<SelectRangesEmptyContent />}
    {...props}
  >
    {SelectMultipleRanges}
  </Input.Item>
);

export interface SelectRangeInputItemProps
  extends Omit<InputItemProps<string, string, SelectRangeProps>, "label"> {}

export const SelectRangeInputItem = (
  props: SelectRangeInputItemProps
): ReactElement => (
  <Input.Item<string, string, SelectRangeProps> label="Range:" {...props}>
    {SelectRange}
  </Input.Item>
);
