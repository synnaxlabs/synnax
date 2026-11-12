// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds, primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Item } from "@/select/Item";
import { Simple, type SingleSimpleProps } from "@/select/Simple";

/** Props for {@link Select}. */
export interface SelectProps extends Omit<
  SingleSimpleProps<string>,
  "value" | "onChange" | "children" | "allowNone" | "resourceName"
> {
  /** The bounds to pick from. */
  data: bounds.Bounds[];
  /** The picked bounds. A value whose ends match none of `data` picks nothing. */
  value: bounds.Bounds;
  onChange: (next: bounds.Bounds) => void;
  /** Shown after the numbers, such as a unit. */
  units?: string;
  /** Singular name of what the bounds are. Defaults to "bounds". */
  resourceName?: string;
}

const toKey = ({ lower, upper }: bounds.Bounds): string => `${lower}:${upper}`;

/** @returns "±10 V" for bounds symmetric about zero, and "0 to 5 V" for others. */
export const format = ({ lower, upper }: bounds.Bounds, units?: string): string => {
  const text = lower === -upper ? `±${Math.abs(upper)}` : `${lower} to ${upper}`;
  return primitive.isNonZero(units) ? `${text} ${units}` : text;
};

/**
 * A dropdown that picks one of a list of bounds, such as the input ranges that a
 * device supports.
 *
 * @example
 * <Bounds.Select data={ranges} value={{ lower: min, upper: max }} onChange={set} />
 */
export const Select = ({
  data,
  value,
  onChange,
  units,
  resourceName = "bounds",
  ...rest
}: SelectProps): ReactElement => {
  const options = [...new Map(data.map((b) => [toKey(b), b])).values()];
  const picked = options.find((b) => toKey(b) === toKey(value));
  const handleChange = (key: string | null) => {
    const next = options.find((b) => toKey(b) === key);
    if (next != null) onChange(next);
  };
  return (
    <Simple<string>
      searchHidden
      resourceName={resourceName}
      value={picked == null ? undefined : toKey(picked)}
      onChange={handleChange}
      {...rest}
    >
      {options.map((b) => (
        <Item key={toKey(b)} itemKey={toKey(b)}>
          {format(b, units)}
        </Item>
      ))}
    </Simple>
  );
};
