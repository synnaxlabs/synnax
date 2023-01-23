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
  <Select.Multiple columns={rangeListColumns} {...props} />
);

export interface SelectRangeProps extends Omit<SelectProps<Range>, "columns"> {}

export const SelectRange = (props: SelectRangeProps): JSX.Element => (
  <Select columns={rangeListColumns} {...props} />
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
