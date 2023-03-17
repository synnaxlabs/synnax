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
} from "@synnaxlabs/pluto";

import { Range } from "../store";

import { rangeListColumns } from "./RangesList";

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

export const SelectMultipleRangesInputItem = (
  props: SelectMultipleRangesInputItemProps
): JSX.Element => (
  <Input.Item<readonly string[], readonly string[], SelectMultipleRangesProps>
    direction="x"
    label="Ranges:"
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
