// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/form/Range.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { bounds, type deep } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { CSS } from "@/css";

type Edge = keyof bounds.Bounds;

interface Held {
  typed: bounds.Bounds;
  /** The stored range when the hold started. */
  stored: bounds.Bounds;
}

const INVALID = "The range must start below its end";

export interface RangeInputProps {
  /** Names the range in the labels of its inputs. */
  label: string;
  value: bounds.Bounds;
  onChange: (next: bounds.Bounds) => void;
}

/**
 * Edits a range as its start and end on one line. The Core refuses a range whose start
 * is not below its end, so the input holds such a range with an error and saves nothing
 * until it is valid. A change to the stored range drops the held one.
 */
export const RangeInput = ({
  label,
  value,
  onChange,
}: RangeInputProps): ReactElement => {
  const [held, setHeld] = useState<Held>();
  const stale = held != null && !bounds.equals(held.stored, value);
  if (stale) setHeld(undefined);
  const holding = held != null && !stale;
  const shown = holding ? held.typed : value;
  const handleChange = (edge: Edge) => (v: number) => {
    const next = { ...shown, [edge]: v };
    if (next.lower >= next.upper) return setHeld({ typed: next, stored: value });
    setHeld(undefined);
    onChange(next);
  };
  const status = holding ? "error" : undefined;
  const tooltip = holding ? INVALID : undefined;
  return (
    <Flex.Box x align="center" gap="small" className={CSS.B("range-input")}>
      <Input.Numeric
        aria-label={`${label} start`}
        value={shown.lower}
        onChange={handleChange("lower")}
        status={status}
        tooltip={tooltip}
      />
      <Text.Text level="small" color={9}>
        to
      </Text.Text>
      <Input.Numeric
        aria-label={`${label} end`}
        value={shown.upper}
        onChange={handleChange("upper")}
        status={status}
        tooltip={tooltip}
      />
    </Flex.Box>
  );
};

export interface RangeFieldProps {
  /** Path to the range within the config of the node. */
  path: deep.Target<bounds.Bounds>;
}

/**
 * The {@link RangeInput} of the range at the path. It has no label: place it in a
 * section titled "Range".
 */
export const RangeField = ({ path }: RangeFieldProps): ReactElement => {
  const { value, onChange } = Form.useField(path);
  return <RangeInput label="Range" value={value} onChange={onChange} />;
};
