// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { Telem } from "@synnaxlabs/pluto";
import { DataType, deep, type xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { z } from "zod";

import {
  type SlaveDevice,
  type StartupParameter,
} from "@/feature/ethercat/device/types";

const DATA_TYPES = [
  DataType.INT8,
  DataType.INT16,
  DataType.INT32,
  DataType.INT64,
  DataType.UINT8,
  DataType.UINT16,
  DataType.UINT32,
  DataType.UINT64,
  DataType.FLOAT32,
  DataType.FLOAT64,
];

const HIDDEN_DATA_TYPES = DataType.ALL.filter(
  (dt) => !DATA_TYPES.some((t) => t.equals(dt)),
);

// The Driver reads values as doubles, which hold whole numbers exactly only up to this.
const EXACT = Number.MAX_SAFE_INTEGER;

const INTEGER_RANGES: Record<string, [number, number]> = {
  int8: [-128, 127],
  int16: [-32768, 32767],
  int32: [-2147483648, 2147483647],
  int64: [-EXACT, EXACT],
  uint8: [0, 255],
  uint16: [0, 65535],
  uint32: [0, 4294967295],
  uint64: [0, EXACT],
};

const FLOAT_MAXES: Record<string, number> = {
  float32: 3.4028234663852886e38,
  float64: Number.MAX_VALUE,
};

/** @returns why the data type cannot hold the value, in the Driver's words, or null. */
const valueError = (dataType: string, value: number): string | null => {
  const range = INTEGER_RANGES[dataType];
  if (range != null) {
    const [min, max] = range;
    if (Number.isInteger(value) && value >= min && value <= max) return null;
    return `must be a whole number from ${min} to ${max}`;
  }
  const max = FLOAT_MAXES[dataType];
  if (max == null || Math.abs(value) <= max) return null;
  return `${value} does not fit in ${dataType}`;
};

const parametersZ = z.array(
  z.object({ dataType: z.string(), value: z.number() }).superRefine((p, ctx) => {
    const message = valueError(p.dataType, p.value);
    if (message != null) ctx.addIssue({ code: "custom", path: ["value"], message });
  }),
);

const FIELDS: (keyof StartupParameter)[] = [
  "index",
  "subIndex",
  "dataType",
  "value",
  "name",
];

const formatHex = (value: number, digits: number): string =>
  `0x${value.toString(16).toUpperCase().padStart(digits, "0")}`;

interface HexCellProps extends Input.TableCellProps<number> {
  digits: number;
}

/** A hexadecimal cell. Text that is not a hex number of at most digits reverts. */
const HexCell = ({ value, onChange, digits, ...rest }: HexCellProps): ReactElement => (
  <Input.Text
    {...rest}
    onlyChangeOnBlur
    value={formatHex(value, digits)}
    onChange={(text) => {
      const match = new RegExp(`^(0x)?([0-9a-f]{1,${digits}})$`, "i").exec(text.trim());
      if (match != null) onChange(Number.parseInt(match[2], 16));
    }}
  />
);

const createRow = (): Input.TableCell[] => [0, 0, DataType.UINT8.toString(), 0, ""];

const PATH = deep.path<SlaveDevice>().properties.startupParameters;

/** Edits the CoE values that the Driver writes to a slave each time the network starts. */
export const StartupParameters = (): ReactElement => {
  const { value, onChange } = Form.useField(PATH);
  const statuses = Form.useStatuses(PATH);
  Form.useCheck(PATH, parametersZ);
  const rows = value.map((p) => FIELDS.map((f) => p[f]));
  const handleChange = (next: Input.TableCell[][]) =>
    onChange(
      next.map(([index, subIndex, dataType, value, name]) => ({
        index: Number(index),
        subIndex: Number(subIndex),
        dataType: String(dataType),
        value: Number(value),
        name: String(name),
      })),
    );
  const cellStatus = ({ x, y }: xy.XY) =>
    statuses.find(({ key }) => key === `${String(PATH)}.${y}.${FIELDS[x]}`);
  return (
    <>
      <Input.Table
        value={rows}
        onChange={handleChange}
        createRow={createRow}
        resourceName="parameter"
        cellStatus={cellStatus}
      >
        <Input.TableColumn name="Index" width={14}>
          {(p) => <HexCell {...p} digits={4} />}
        </Input.TableColumn>
        <Input.TableColumn name="Subindex" width={12}>
          {(p) => <HexCell {...p} digits={2} />}
        </Input.TableColumn>
        <Input.TableColumn name="Data type" type="string" width={18}>
          {({ value, onChange, variant, "aria-label": ariaLabel }) => (
            <Telem.SelectDataType
              value={value}
              onChange={onChange}
              hideDataTypes={HIDDEN_DATA_TYPES}
              allowNone={false}
              triggerProps={{ "aria-label": ariaLabel, variant }}
            />
          )}
        </Input.TableColumn>
        <Input.TableColumn name="Value" />
        <Input.TableColumn name="Name" type="string">
          {(p) => <Input.Text {...p} placeholder="Filter settings" />}
        </Input.TableColumn>
      </Input.Table>
      <Text.Text level="small" color={9}>
        The Driver writes these CoE values to the device each time the network starts,
        before the device exchanges data.
      </Text.Text>
    </>
  );
};
