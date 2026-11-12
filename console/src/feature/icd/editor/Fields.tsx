// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { deep, uuid } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

type Field = icd.BinaryField | icd.TextField;
type Encoding = "binary" | icd.TextFieldType;
type FieldOf<E extends Encoding> = E extends "binary"
  ? icd.BinaryField
  : Extract<icd.TextField, { encoding: E }>;

const encodingOf = (field: Field): Encoding =>
  "encoding" in field ? field.encoding : "binary";

/** Marks an absent multiplexor in a select cell. */
const NONE = "none";

type Render = "text" | "number" | "byteOrder" | "type" | "multiplexor";

interface Column<F extends Field> {
  name: string;
  render: Render;
  /** The column's width in rem. Omitted, it shares what the others leave. */
  width?: number;
  get(field: F): Input.TableCell;
  set(field: F, cell: Input.TableCell): F;
}

const name = <F extends Field>(): Column<F> => ({
  name: "Name",
  render: "text",
  get: ({ name }) => name,
  set: (f, cell) => ({ ...f, name: String(cell) }),
});

const scale = <F extends Field>(): Column<F> => ({
  name: "Scale",
  render: "number",
  width: 11,
  get: ({ scale }) => scale,
  set: (f, cell) => ({ ...f, scale: Number(cell) }),
});

const offset = <F extends Field>(): Column<F> => ({
  name: "Offset",
  render: "number",
  width: 11,
  get: ({ offset }) => offset,
  set: (f, cell) => ({ ...f, offset: Number(cell) }),
});

const units = <F extends Field>(): Column<F> => ({
  name: "Units",
  render: "text",
  width: 11,
  get: ({ units }) => units,
  set: (f, cell) => ({ ...f, units: String(cell) }),
});

const multiplexor = <F extends Field>(): Column<F> => ({
  name: "Multiplexor",
  render: "multiplexor",
  width: 18,
  get: ({ multiplexor }) => multiplexor ?? NONE,
  set: (f, cell) => ({ ...f, multiplexor: cell === NONE ? undefined : String(cell) }),
});

const parseIntegers = (text: string): number[] =>
  text
    .split(/[\s,]+/)
    .filter((part) => part !== "")
    .map(Number)
    .filter(Number.isInteger);

const multiplexValues = <F extends Field>(): Column<F> => ({
  name: "Multiplex values",
  render: "text",
  width: 21,
  get: ({ multiplexValues }) => multiplexValues.join(", "),
  set: (f, cell) => ({ ...f, multiplexValues: parseIntegers(String(cell)) }),
});

const conversion = <F extends Field>(): Column<F>[] => [scale(), offset(), units()];

type ValueType = "unsigned" | "signed" | "float";

const VALUE_TYPE_NAMES: Record<ValueType, string> = {
  unsigned: "Unsigned",
  signed: "Signed",
  float: "Float",
};

const BYTE_ORDER_NAMES: Record<icd.ByteOrder, string> = {
  little_endian: "Little endian",
  big_endian: "Big endian",
};

const valueTypeOf = ({ signed, float }: icd.BinaryField): ValueType => {
  if (float) return "float";
  return signed ? "signed" : "unsigned";
};

const COLUMNS: { [E in Encoding]: Column<FieldOf<E>>[] } = {
  binary: [
    name(),
    {
      name: "Start bit",
      render: "number",
      width: 14,
      get: ({ startBit }) => startBit,
      set: (f, cell) => ({ ...f, startBit: Number(cell) }),
    },
    {
      name: "Bit length",
      render: "number",
      width: 14,
      get: ({ bitLength }) => bitLength,
      set: (f, cell) => ({ ...f, bitLength: Number(cell) }),
    },
    {
      name: "Byte order",
      render: "byteOrder",
      width: 22,
      get: ({ byteOrder }) => byteOrder,
      set: (f, cell) => ({ ...f, byteOrder: cell as icd.ByteOrder }),
    },
    {
      name: "Type",
      render: "type",
      width: 20,
      get: valueTypeOf,
      set: (f, cell) => ({ ...f, signed: cell === "signed", float: cell === "float" }),
    },
    ...conversion<icd.BinaryField>(),
    multiplexor(),
    multiplexValues(),
  ],
  delimited: [
    name(),
    {
      name: "Position",
      render: "number",
      width: 13,
      get: ({ position }) => position,
      set: (f, cell) => ({ ...f, position: Number(cell) }),
    },
    ...conversion<icd.DelimitedTextField>(),
  ],
  tagged: [
    name(),
    {
      name: "Tag",
      render: "text",
      width: 14,
      get: ({ tag }) => tag,
      set: (f, cell) => ({ ...f, tag: String(cell) }),
    },
    ...conversion<icd.TaggedTextField>(),
  ],
};

const createField = (encoding: Encoding, name: string = ""): Field => {
  const base = {
    key: uuid.create(),
    name,
    scale: 1,
    offset: 0,
    units: "",
    multiplexValues: [],
  };
  switch (encoding) {
    case "binary":
      return {
        ...base,
        startBit: 0,
        bitLength: 8,
        byteOrder: "little_endian",
        signed: false,
        float: false,
      };
    case "delimited":
      return { ...base, encoding, position: 0 };
    case "tagged":
      return { ...base, encoding, tag: "" };
  }
};

/** @returns the payload bit just past the field. */
const endBit = ({ startBit, bitLength, byteOrder }: icd.BinaryField): number => {
  if (byteOrder === "little_endian") return startBit + bitLength;
  // A big-endian field runs down from its start bit, then into the bytes after it.
  const rest = Math.max(0, bitLength - (startBit % 8) - 1);
  return (Math.floor(startBit / 8) + Math.ceil(rest / 8) + 1) * 8;
};

const nextStartBit = (fields: Field[]): number =>
  fields.reduce((bit, f) => ("startBit" in f ? Math.max(bit, endBit(f)) : bit), 0);

const uniqueFieldName = (names: string[]): string => {
  const taken = new Set(names);
  let i = names.length + 1;
  while (taken.has(`Field ${i}`)) i++;
  return `Field ${i}`;
};

/**
 * Rebuilds the fields a grid shows from its rows. Edits keep row positions, so each row
 * updates the field at its index and keeps that field's key, which other fields and
 * tasks reference.
 */
const fieldsFromRows = (
  encoding: Encoding,
  columns: Column<Field>[],
  prevFields: Field[],
  rows: Input.TableCell[][],
): Field[] =>
  rows.map((row, i) =>
    columns.reduce<Field>(
      (field, column, j) => column.set(field, row[j]),
      prevFields[i] ?? createField(encoding),
    ),
  );

/** Writes the fields of one encoding back into their slots among all the fields. */
const mergeFields = (all: Field[], encoding: Encoding, updated: Field[]): Field[] => {
  let i = 0;
  const merged = all.flatMap((field) => {
    if (encodingOf(field) !== encoding) return [field];
    return i < updated.length ? [updated[i++]] : [];
  });
  return [...merged, ...updated.slice(i)];
};

const selectColumn = (
  { name, width }: Column<Field>,
  index: number,
  resourceName: string,
  names: Record<string, string>,
): ReactElement<Input.TableColumnProps> => (
  <Input.TableColumn key={index} name={name} width={width} type="string">
    {({ value, onChange, preview, variant, "aria-label": ariaLabel }) => (
      <Select.Simple<string>
        resourceName={resourceName}
        value={value}
        onChange={onChange}
        preview={preview}
        variant="floating"
        triggerProps={{ variant, "aria-label": ariaLabel }}
      >
        {Object.entries(names).map(([key, label]) => (
          <Select.Item key={key} itemKey={key}>
            {label}
          </Select.Item>
        ))}
      </Select.Simple>
    )}
  </Input.TableColumn>
);

const renderColumn = (
  column: Column<Field>,
  index: number,
  fields: Field[],
): ReactElement<Input.TableColumnProps> => {
  const { name, render, width } = column;
  switch (render) {
    case "number":
      return <Input.TableColumn key={index} name={name} width={width} />;
    case "text":
      return <Input.TableColumn key={index} name={name} width={width} type="string" />;
    case "type":
      return selectColumn(column, index, "type", VALUE_TYPE_NAMES);
    case "byteOrder":
      return selectColumn(column, index, "byte order", BYTE_ORDER_NAMES);
    case "multiplexor":
      return selectColumn(column, index, "multiplexor", {
        [NONE]: "None",
        ...Object.fromEntries(fields.map(({ key, name }) => [key, name])),
      });
  }
};

const matchOwnField = (
  key: string | undefined,
  path: deep.Path<Field[]>,
  field: Field,
): boolean => key != null && deep.pathsMatch(key, String(path[field.key]));

export interface FieldsProps {
  /** The form path of the message's fields. */
  path: deep.Path<Field[]>;
  encoding: Encoding;
}

/** Edits the fields of one encoding in a message payload as a grid. */
export const Fields = ({ path, encoding }: FieldsProps): ReactElement => {
  const { set, mode } = Form.useContext();
  const fields = Form.useFieldValue(path);
  const status = Form.useStatuses(path).find(({ key }) =>
    fields.some((f) => encodingOf(f) === encoding && matchOwnField(key, path, f)),
  );
  const columns = COLUMNS[encoding] as Column<Field>[];
  const own = fields.filter((f) => encodingOf(f) === encoding);
  const rows = own.map((f) => columns.map((c) => c.get(f)));
  const handleChange = (next: Input.TableCell[][]) =>
    set(
      path,
      mergeFields(fields, encoding, fieldsFromRows(encoding, columns, own, next)),
    );
  const handleRemove = (index: number) =>
    set(
      path,
      fields.filter(({ key }) => key !== own[index].key),
    );
  // A paste adds several rows before any is stored. Each grid's first column is the
  // name.
  const createRow = (prevRows: Input.TableCell[][]) => {
    const added = prevRows.slice(own.length).map(([name]) => String(name));
    const field = createField(
      encoding,
      uniqueFieldName([...fields.map((f) => f.name), ...added]),
    );
    if ("startBit" in field)
      field.startBit = nextStartBit(fieldsFromRows(encoding, columns, own, prevRows));
    return columns.map((c) => c.get(field));
  };
  const failing = own.findIndex((f) => matchOwnField(status?.key, path, f));
  let helpText = status?.message;
  if (failing !== -1) helpText = `Field ${failing + 1}: ${helpText}`;
  return (
    <Flex.Box y gap="small" className={CSS.BE("icd-editor", "fields")}>
      <Input.Table
        value={rows}
        onChange={handleChange}
        onRemove={handleRemove}
        createRow={createRow}
        rowLabel={(i) => (i + 1).toString()}
        preview={mode === "preview"}
      >
        {columns.map((column, i) => renderColumn(column, i, fields))}
      </Input.Table>
      {helpText != null && (
        <Input.HelpText variant={status?.variant}>{helpText}</Input.HelpText>
      )}
    </Flex.Box>
  );
};
