// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type library } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { deep, uuid } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";

import { useNestedStatus } from "@/feature/library/editor/useNestedStatus";

type Encoding = library.FieldType;
type FieldOf<E extends Encoding> = Extract<library.Field, { encoding: E }>;

/** Marks an absent enumeration or multiplexor in a select cell. */
const NONE = "none";

type Render =
  "text" | "number" | "byteOrder" | "boolean" | "enumeration" | "multiplexor";

interface Column<F extends library.Field> {
  name: string;
  render: Render;
  get(field: F): Input.TableCell;
  set(field: F, cell: Input.TableCell): F;
}

const name = <F extends library.Field>(): Column<F> => ({
  name: "Name",
  render: "text",
  get: ({ name }) => name,
  set: (f, cell) => ({ ...f, name: String(cell) }),
});

const scale = <F extends library.Field>(): Column<F> => ({
  name: "Scale",
  render: "number",
  get: ({ scale }) => scale,
  set: (f, cell) => ({ ...f, scale: Number(cell) }),
});

const offset = <F extends library.Field>(): Column<F> => ({
  name: "Offset",
  render: "number",
  get: ({ offset }) => offset,
  set: (f, cell) => ({ ...f, offset: Number(cell) }),
});

const units = <F extends library.Field>(): Column<F> => ({
  name: "Units",
  render: "text",
  get: ({ units }) => units,
  set: (f, cell) => ({ ...f, units: String(cell) }),
});

const enumeration = <F extends library.Field>(): Column<F> => ({
  name: "Enumeration",
  render: "enumeration",
  get: ({ enumeration }) => enumeration ?? NONE,
  set: (f, cell) => ({ ...f, enumeration: cell === NONE ? undefined : String(cell) }),
});

const multiplexor = <F extends library.Field>(): Column<F> => ({
  name: "Multiplexor",
  render: "multiplexor",
  get: ({ multiplexor }) => multiplexor ?? NONE,
  set: (f, cell) => ({ ...f, multiplexor: cell === NONE ? undefined : String(cell) }),
});

const parseIntegers = (text: string): number[] =>
  text
    .split(/[\s,]+/)
    .filter((part) => part !== "")
    .map(Number)
    .filter(Number.isInteger);

const multiplexValues = <F extends library.Field>(): Column<F> => ({
  name: "Multiplex values",
  render: "text",
  get: ({ multiplexValues }) => multiplexValues.join(", "),
  set: (f, cell) => ({ ...f, multiplexValues: parseIntegers(String(cell)) }),
});

const shared = <F extends library.Field>(): Column<F>[] => [
  scale(),
  offset(),
  units(),
  enumeration(),
  multiplexor(),
  multiplexValues(),
];

const COLUMNS: { [E in Encoding]: Column<FieldOf<E>>[] } = {
  binary: [
    name(),
    {
      name: "Start bit",
      render: "number",
      get: ({ startBit }) => startBit,
      set: (f, cell) => ({ ...f, startBit: Number(cell) }),
    },
    {
      name: "Bit length",
      render: "number",
      get: ({ bitLength }) => bitLength,
      set: (f, cell) => ({ ...f, bitLength: Number(cell) }),
    },
    {
      name: "Byte order",
      render: "byteOrder",
      get: ({ byteOrder }) => byteOrder,
      set: (f, cell) => ({ ...f, byteOrder: cell as library.ByteOrder }),
    },
    {
      name: "Signed",
      render: "boolean",
      get: ({ signed }) => Number(signed),
      set: (f, cell) => ({ ...f, signed: cell === 1 }),
    },
    {
      name: "Float",
      render: "boolean",
      get: ({ float }) => Number(float),
      set: (f, cell) => ({ ...f, float: cell === 1 }),
    },
    ...shared<library.BinaryField>(),
  ],
  delimited: [
    name(),
    {
      name: "Position",
      render: "number",
      get: ({ position }) => position,
      set: (f, cell) => ({ ...f, position: Number(cell) }),
    },
    ...shared<library.DelimitedField>(),
  ],
  tagged: [
    name(),
    {
      name: "Tag",
      render: "text",
      get: ({ tag }) => tag,
      set: (f, cell) => ({ ...f, tag: String(cell) }),
    },
    ...shared<library.TaggedField>(),
  ],
};

const createField = (encoding: Encoding, name: string = ""): library.Field => {
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
        encoding,
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

const uniqueFieldName = (fields: library.Field[]): string => {
  const names = new Set(fields.map(({ name }) => name));
  let i = fields.length + 1;
  while (names.has(`Field ${i}`)) i++;
  return `Field ${i}`;
};

/**
 * Rebuilds the fields a grid shows from its rows. A removal keeps the rows it does not
 * touch, so a shorter grid matches rows to fields by identity; every other edit keeps
 * row positions. Matching keeps each field's key, which other fields and tasks
 * reference.
 */
const fieldsFromRows = (
  encoding: Encoding,
  columns: Column<library.Field>[],
  prevRows: Input.TableCell[][],
  prevFields: library.Field[],
  rows: Input.TableCell[][],
): library.Field[] => {
  const removed = rows.length < prevRows.length;
  return rows.map((row, i) => {
    const prev = prevFields[removed ? prevRows.indexOf(row) : i];
    return columns.reduce<library.Field>(
      (field, column, j) => column.set(field, row[j]),
      prev ?? createField(encoding),
    );
  });
};

/** Writes the fields of one encoding back into their slots among all the fields. */
const mergeFields = (
  all: library.Field[],
  encoding: Encoding,
  updated: library.Field[],
): library.Field[] => {
  let i = 0;
  const merged = all.flatMap((field) => {
    if (field.encoding !== encoding) return [field];
    return i < updated.length ? [updated[i++]] : [];
  });
  return [...merged, ...updated.slice(i)];
};

interface CellContext {
  enums: library.EnumEntry[];
  fields: library.Field[];
}

const renderColumn = (
  column: Column<library.Field>,
  index: number,
  { enums, fields }: CellContext,
): ReactElement<Input.TableColumnProps> => {
  const { name, render } = column;
  switch (render) {
    case "number":
      return <Input.TableColumn key={index} name={name} />;
    case "text":
      return <Input.TableColumn key={index} name={name} type="string" />;
    case "boolean":
      return (
        <Input.TableColumn key={index} name={name}>
          {({ value, onChange, preview, "aria-label": ariaLabel }) => (
            <Input.Checkbox
              value={value === 1}
              onChange={(checked) => onChange(checked ? 1 : 0)}
              preview={preview}
              aria-label={ariaLabel}
            />
          )}
        </Input.TableColumn>
      );
    case "byteOrder":
      return (
        <Input.TableColumn key={index} name={name} type="string">
          {({ value, onChange, preview }) => (
            <Select.Simple<string>
              resourceName="byte order"
              value={value}
              onChange={onChange}
              preview={preview}
            >
              <Select.Item itemKey="little_endian">Little endian</Select.Item>
              <Select.Item itemKey="big_endian">Big endian</Select.Item>
            </Select.Simple>
          )}
        </Input.TableColumn>
      );
    case "enumeration":
      return (
        <Input.TableColumn key={index} name={name} type="string">
          {({ value, onChange, preview }) => (
            <Select.Simple<string>
              resourceName="enumeration"
              value={value}
              onChange={onChange}
              preview={preview}
            >
              <Select.Item itemKey={NONE}>None</Select.Item>
              {enums.map(({ key, name }) => (
                <Select.Item key={key} itemKey={key}>
                  {name}
                </Select.Item>
              ))}
            </Select.Simple>
          )}
        </Input.TableColumn>
      );
    case "multiplexor":
      return (
        <Input.TableColumn key={index} name={name} type="string">
          {({ value, onChange, preview }) => (
            <Select.Simple<string>
              resourceName="multiplexor"
              value={value}
              onChange={onChange}
              preview={preview}
            >
              <Select.Item itemKey={NONE}>None</Select.Item>
              {fields.map(({ key, name }) => (
                <Select.Item key={key} itemKey={key}>
                  {name}
                </Select.Item>
              ))}
            </Select.Simple>
          )}
        </Input.TableColumn>
      );
  }
};

const matchOwnField = (
  key: string | undefined,
  fieldsPath: string,
  field: library.Field,
): boolean => key != null && deep.pathsMatch(key, `${fieldsPath}.${field.key}`);

export interface FieldsProps {
  /** The form path of the message entry. */
  path: string;
  encoding: Encoding;
  label: string;
}

/** Edits the fields of one encoding in a message as a grid. */
export const Fields = ({ path, encoding, label }: FieldsProps): ReactElement => {
  const fieldsPath = `${path}.fields`;
  const { set } = Form.useContext();
  const fields = Form.useFieldValue<library.Field[]>(fieldsPath);
  const entries = Form.useFieldValue<library.Entry[]>("entries");
  const enums = useMemo(
    () => entries.filter((e): e is library.EnumEntry => e.kind === "enum"),
    [entries],
  );
  const columns = COLUMNS[encoding] as Column<library.Field>[];
  const own = useMemo(
    () => fields.filter((f) => f.encoding === encoding),
    [fields, encoding],
  );
  const rows = useMemo(
    () => own.map((f) => columns.map((c) => c.get(f))),
    [own, columns],
  );
  const handleChange = (next: Input.TableCell[][]) =>
    set(
      fieldsPath,
      mergeFields(fields, encoding, fieldsFromRows(encoding, columns, rows, own, next)),
    );
  const createRow = () =>
    columns.map((c) => c.get(createField(encoding, uniqueFieldName(fields))));
  const matchOwn = useCallback(
    (key: string) => own.some((f) => matchOwnField(key, fieldsPath, f)),
    [own, fieldsPath],
  );
  const status = useNestedStatus(fieldsPath, matchOwn);
  const failing = own.findIndex((f) => matchOwnField(status?.key, fieldsPath, f));
  let helpText = status?.message;
  if (failing !== -1) helpText = `Field ${failing + 1}: ${helpText}`;
  return (
    <Input.Item label={label} padHelpText helpText={helpText} status={status?.variant}>
      <Input.Table
        value={rows}
        onChange={handleChange}
        createRow={createRow}
        rowLabel={(i) => (i + 1).toString()}
      >
        {columns.map((column, i) => renderColumn(column, i, { enums, fields }))}
      </Input.Table>
    </Input.Item>
  );
};
