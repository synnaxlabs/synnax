// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ethercat, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Telem } from "@synnaxlabs/pluto";
import { DataType, deep, type xy } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { z } from "zod";

import {
  mergeStartupParameters,
  parseStartupCommands,
  type StartupCommands,
} from "@/feature/ethercat/device/esi";
import { sameAddress, visibleFields } from "@/feature/ethercat/device/settings";
import { type SlaveDevice } from "@/feature/ethercat/device/types";
import { Runtime } from "@/platform/runtime";

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

const FIELDS: (keyof ethercat.StartupParameter)[] = [
  "index",
  "subIndex",
  "dataType",
  "value",
  "name",
];

/** @returns the number that text writes in hex with at most digits, or null. */
const parseHex = (text: string, digits: number): number | null => {
  const match = new RegExp(`^(0x)?([0-9a-f]{1,${digits}})$`, "i").exec(text.trim());
  return match == null ? null : Number.parseInt(match[2], 16);
};

const INDEX_DIGITS = 4;
const SUB_INDEX_DIGITS = 2;

const parseIndex = (text: string): number | null => parseHex(text, INDEX_DIGITS);

const parseSubIndex = (text: string): number | null => parseHex(text, SUB_INDEX_DIGITS);

// The IEC 61131-3 names that TwinCAT and ESI files give CoE objects.
const IEC_DATA_TYPES: Record<string, DataType> = {
  SINT: DataType.INT8,
  INT: DataType.INT16,
  DINT: DataType.INT32,
  LINT: DataType.INT64,
  USINT: DataType.UINT8,
  UINT: DataType.UINT16,
  UDINT: DataType.UINT32,
  ULINT: DataType.UINT64,
  BYTE: DataType.UINT8,
  WORD: DataType.UINT16,
  DWORD: DataType.UINT32,
  LWORD: DataType.UINT64,
  REAL: DataType.FLOAT32,
  LREAL: DataType.FLOAT64,
};

/** @returns the data type that text names, in Synnax or IEC 61131-3 terms, or null. */
const parseDataType = (text: string): string | null => {
  const name = text.trim();
  const dataType =
    IEC_DATA_TYPES[name.toUpperCase()] ??
    DATA_TYPES.find((dt) => dt.toString() === name.toLowerCase());
  return dataType?.toString() ?? null;
};

interface HexCellProps extends Input.TableCellProps<number> {
  digits: number;
}

/** A hexadecimal cell. Text that is not a hex number of at most digits reverts. */
const HexCell = ({ value, onChange, digits, ...rest }: HexCellProps): ReactElement => (
  <Input.Text
    {...rest}
    onlyChangeOnBlur
    value={ethercat.formatHex(value, digits)}
    onChange={(text) => {
      const parsed = parseHex(text, digits);
      if (parsed != null) onChange(parsed);
    }}
  />
);

const createRow = (): Input.TableCell[] => [0, 0, DataType.UINT8.toString(), 0, ""];

const PROPERTIES = deep.path<SlaveDevice>().properties;
const PATH = PROPERTIES.startupParameters;

const IMPORT_TITLE = "Import from ESI file";

/** @returns a status that tells what an import added, and what it left out. */
const describeImport = ({
  device,
  parameters,
  unwritableCount,
  modeCommands,
}: StartupCommands): status.Status => {
  const plural = parameters.length === 1 ? "" : "s";
  const notes: string[] = [];
  if (modeCommands.length > 0)
    notes.push(
      `Left out the mode of the device at ${ethercat.formatAddress(modeCommands[0])}.`,
    );
  if (unwritableCount > 0)
    notes.push(`Skipped ${unwritableCount} that the Driver cannot write.`);
  return status.create({
    variant: "success",
    message: `Imported ${parameters.length} startup parameter${plural} for ${device}`,
    description: notes.join(" "),
  });
};

/**
 * Edits the CoE values that the Driver writes to a slave each time the network starts.
 * A parameter with no name shows the name of the shown catalog setting at its address.
 */
export const StartupParameters = (): ReactElement => {
  Form.useCheck(PATH, parametersZ);
  const { value: parameters, onChange } = Form.useField(PATH);
  const { get } = Form.useContext();
  const statuses = Form.useStatuses(PATH);
  const fields = Form.useFieldValue(PROPERTIES.settings, { optional: true })?.device;
  const shown = fields == null ? [] : visibleFields(fields, parameters);
  const [result, setResult] = useState<status.Status | null>(null);
  const handleImport = async () => {
    try {
      const file = await Runtime.pickFiles({ title: IMPORT_TITLE, extension: "xml" });
      if (file == null) return;
      const bytes = await Runtime.toBytes(await file.read());
      const commands = parseStartupCommands(bytes, get(PROPERTIES).value);
      onChange(mergeStartupParameters(get(PATH).value, commands.parameters));
      setResult(describeImport(commands));
    } catch (e) {
      setResult(status.fromException(e, "Failed to import ESI file"));
    }
  };
  const settingName = (address: ethercat.StartupParameter): string =>
    shown.find((f) => sameAddress(f, address))?.name ?? "";
  const rows = parameters.map((p) =>
    FIELDS.map((f) => (f === "name" && p.name === "" ? settingName(p) : p[f])),
  );
  const handleChange = (next: Input.TableCell[][]) =>
    onChange(
      next.map(([index, subIndex, dataType, value, name], i) => {
        const parameter = {
          index: Number(index),
          subIndex: Number(subIndex),
          dataType: String(dataType),
          value: Number(value),
          name: String(name),
        };
        const before = parameters.at(i);
        // A cell still showing the catalog name of an unnamed parameter stays unnamed.
        const shownBefore =
          before?.name === "" && parameter.name === settingName(before);
        if (shownBefore || parameter.name === settingName(parameter))
          parameter.name = "";
        return parameter;
      }),
    );
  const cellStatus = ({ x, y }: xy.XY) =>
    statuses.find(({ key }) => key === `${String(PATH)}.${y}.${FIELDS[x]}`);
  return (
    <>
      <Input.Table
        value={rows}
        onChange={handleChange}
        onRemove={(i) => onChange(parameters.filter((_, j) => j !== i))}
        createRow={createRow}
        resourceName="parameter"
        cellStatus={cellStatus}
      >
        <Input.TableColumn name="Index" width={14} parse={parseIndex}>
          {(p) => <HexCell {...p} digits={INDEX_DIGITS} />}
        </Input.TableColumn>
        <Input.TableColumn name="Subindex" width={12} parse={parseSubIndex}>
          {(p) => <HexCell {...p} digits={SUB_INDEX_DIGITS} />}
        </Input.TableColumn>
        <Input.TableColumn
          name="Data type"
          type="string"
          width={18}
          parse={parseDataType}
        >
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
      <Button.Button
        variant="text"
        size="small"
        textColor={9}
        onClick={() => void handleImport()}
      >
        <Icon.Import />
        {IMPORT_TITLE}
      </Button.Button>
      {result != null && <Status.Summary status={result} level="small" />}
    </>
  );
};
