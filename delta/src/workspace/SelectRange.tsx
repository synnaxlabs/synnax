// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Input, Status, Button, Select, Align } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { rangeWindowLayout } from "@/workspace/DefineRange";
import { type Range } from "@/workspace/range";
import { rangeListColumns } from "@/workspace/RangesList";

export interface SelectMultipleRangesProps
  extends Omit<Select.MultipleProps<string, Range>, "columns"> {}

export const SelectMultipleRanges = (
  props: SelectMultipleRangesProps
): ReactElement => (
  <Select.Multiple columns={rangeListColumns} tagKey="name" {...props} />
);

export interface SelectSingleRangeProps
  extends Omit<Select.SingleProps<string, Range>, "columns"> {}

export const SelectRange = (props: SelectSingleRangeProps): ReactElement => (
  <Select.Single columns={rangeListColumns} {...props} tagKey="name" />
);

export interface SelectMultipleRangesInputItemProps
  extends Omit<
    Input.ItemProps<string[], string[], SelectMultipleRangesProps>,
    "label"
  > {}

const SelectRangesEmptyContent = (): ReactElement => {
  const newLayout = Layout.usePlacer();
  return (
    <Align.Center style={{ height: 150 }} direction="x">
      <Status.Text variant="disabled" hideIcon>
        No Ranges:
      </Status.Text>
      <Button.Button
        variant="outlined"
        onClick={() => {
          newLayout(rangeWindowLayout);
        }}
      >
        Define a Range
      </Button.Button>
    </Align.Center>
  );
};

export const SelectMultipleRangesInputItem = (
  props: SelectMultipleRangesInputItemProps
): ReactElement => (
  <Input.Item<string[], string[], SelectMultipleRangesProps>
    direction="x"
    label="Ranges"
    emptyContent={<SelectRangesEmptyContent />}
    {...props}
  >
    {SelectMultipleRanges}
  </Input.Item>
);

export interface SelectRangeInputItemProps
  extends Omit<Input.ItemProps<string, string, SelectSingleRangeProps>, "label"> {}

export const SelectRangeInputItem = (
  props: SelectRangeInputItemProps
): ReactElement => (
  <Input.Item<string, string, SelectSingleRangeProps> label="Range:" {...props}>
    {SelectRange}
  </Input.Item>
);
