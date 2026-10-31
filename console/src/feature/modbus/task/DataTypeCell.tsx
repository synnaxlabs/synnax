// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Telem } from "@synnaxlabs/pluto";
import { DataType } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { SHADOW_TRIGGER_PROPS } from "@/feature/modbus/task/Register";
import {
  BIT_BOUNDS,
  type Channel,
  STRING_LENGTH_BOUNDS,
} from "@/feature/modbus/task/types";
import { CSS } from "@/platform/css";

interface DataTypeFieldProps {
  path: string;
  hideDataTypes: DataType[];
}

const DataTypeField = ({
  path,
  hideDataTypes,
}: DataTypeFieldProps): ReactElement | null => {
  const field = Form.useField<string>(`${path}.dataType`);
  if (field == null) return null;
  return (
    <Telem.SelectDataType
      value={field.value}
      onChange={field.onChange}
      preview={field.preview}
      hideDataTypes={hideDataTypes}
      location="bottom"
      triggerProps={SHADOW_TRIGGER_PROPS}
    />
  );
};

interface NumberFieldProps {
  path: string;
  label: string;
  bounds: { lower: number; upper: number };
  /** The shorter name in front of the number, when the cell already gives context. */
  prefix?: string;
}

// A whole number in a cell, named by a prefix in front of it.
const NumberField = ({
  path,
  label,
  bounds,
  prefix = label,
}: NumberFieldProps): ReactElement => (
  <Form.NumericField
    path={path}
    showLabel={false}
    showHelpText={false}
    inputProps={{
      variant: "shadow",
      showDragHandle: false,
      bounds,
      startContent: prefix,
      "aria-label": label,
    }}
  />
);

export interface DataTypeCellProps {
  /** The channel's form path. */
  path: string;
  channel: Channel;
  /** The data types that the task cannot use. */
  hideDataTypes: DataType[];
}

/**
 * The data type of a register channel, and a string's length. A register bit has no
 * data type, so the cell holds its bit. A bit channel has neither, so the cell is
 * empty.
 */
export const DataTypeCell = ({
  path,
  channel,
  hideDataTypes,
}: DataTypeCellProps): ReactElement | null => {
  if ("bit" in channel)
    return <NumberField path={`${path}.bit`} label="Bit" bounds={BIT_BOUNDS} />;
  if (!("dataType" in channel)) return null;
  return (
    <Flex.Box x align="center" className={CSS.BE("modbus", "data-type")}>
      <DataTypeField path={path} hideDataTypes={hideDataTypes} />
      {new DataType(channel.dataType).equals(DataType.STRING) && (
        <NumberField
          path={`${path}.stringLength`}
          label="String length"
          prefix="Length"
          bounds={STRING_LENGTH_BOUNDS}
        />
      )}
    </Flex.Box>
  );
};
