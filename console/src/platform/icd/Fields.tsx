// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/icd/Fields.css";

import { type icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { DataType, deep, payload, type record, uuid } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { EnumValues } from "@/platform/icd/EnumValues";
import { Optional, Options, showsDefault } from "@/platform/icd/Options";
import { Task } from "@/platform/task";

type Field = icd.BinaryField | icd.TextField | icd.JSONField;
type Encoding = "binary" | icd.TextFieldType | "json";
type FieldOf<E extends Encoding> = E extends "binary"
  ? icd.BinaryField
  : E extends "json"
    ? icd.JSONField
    : Extract<icd.TextField, { encoding: E }>;

const encodingOf = (field: Field): Encoding => {
  if ("encoding" in field) return field.encoding;
  return "type" in field ? "json" : "binary";
};

/** Marks an absent multiplexor in a select cell. */
const NONE = "none";

/** Renders a cell that edits its field through the form, given the field's path. */
type Cell<F extends Field> = (
  field: F,
  path: deep.Path<F>,
  props: Input.TableCellProps<string>,
) => ReactElement;

type Render<F extends Field> =
  "text" | "number" | "byteOrder" | "type" | "jsonType" | "multiplexor" | Cell<F>;

interface Column<F extends Field> {
  name: string;
  render: Render<F>;
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

const multiplexor: Column<icd.BinaryField> = {
  name: "Multiplexor",
  render: "multiplexor",
  width: 18,
  get: ({ multiplexor }) => multiplexor ?? NONE,
  set: (f, cell) => ({ ...f, multiplexor: cell === NONE ? undefined : String(cell) }),
};

const parseIntegers = (text: string): number[] =>
  text
    .split(/[\s,]+/)
    .filter((part) => part !== "")
    .map(Number)
    .filter(Number.isInteger);

const multiplexValues: Column<icd.BinaryField> = {
  name: "Multiplex values",
  render: "text",
  width: 21,
  get: ({ multiplexValues }) => multiplexValues.join(", "),
  set: (f, cell) => ({ ...f, multiplexValues: parseIntegers(String(cell)) }),
};

const conversion = <F extends Field>(): Column<F>[] => [scale(), offset(), units()];

/** The kinds a binary field can hold. */
type NumberKind = Extract<payload.Kind, "unsigned" | "integer" | "float">;

const NUMBER_KIND_NAMES: Record<NumberKind, string> = {
  unsigned: "Unsigned integer",
  integer: "Signed integer",
  float: "Float",
};

const BYTE_ORDER_NAMES: Record<icd.ByteOrder, string> = {
  little_endian: "Little endian",
  big_endian: "Big endian",
};

/** A JSON field type, with a timestamp named by its format. */
type JSONType = Exclude<payload.Kind, "timestamp"> | payload.TimeFormat;

const isTimeFormat = (type: JSONType): type is payload.TimeFormat =>
  payload.timeFormatZ.safeParse(type).success;

const isNumberKind = (type: JSONType): type is NumberKind => type in NUMBER_KIND_NAMES;

const JSON_TYPE_NAMES: Record<JSONType, string> = {
  ...NUMBER_KIND_NAMES,
  string: "String",
  boolean: "Boolean",
  json: "JSON",
  uuid: "UUID",
  bytes: "Bytes (base64)",
  iso8601: "Timestamp (ISO 8601)",
  unix_sec: "Timestamp (s)",
  unix_ms: "Timestamp (ms)",
  unix_us: "Timestamp (µs)",
  unix_ns: "Timestamp (ns)",
};

const jsonTypeOf = (field: icd.JSONField): JSONType =>
  field.type === "timestamp" ? field.timeFormat : field.type;

/** The widest data type of each number kind, which holds every value of the kind. */
const NUMBER_DATA_TYPES: Record<NumberKind, string> = {
  float: DataType.FLOAT64.toString(),
  integer: DataType.INT64.toString(),
  unsigned: DataType.UINT64.toString(),
};

/** @returns the enum mapping and default of a number field, as a read field holds. */
const numberOptionsOf = (
  field: icd.JSONField,
): Task.Payload.ReadOptions | undefined => {
  switch (field.type) {
    case "float":
      return { type: field.type, enumValues: field.enumValues, fill: field.default };
    case "integer":
      return { type: field.type, enumValues: field.enumValues, fill: field.default };
    case "unsigned":
      return { type: field.type, enumValues: field.enumValues, fill: field.default };
    default:
      return undefined;
  }
};

/**
 * Only a number field scales, so a field that stops being one drops its scaling. A
 * number field that becomes another number kind keeps each mapping and the default
 * that the new kind holds exactly.
 */
const withJSONType = (field: icd.JSONField, type: JSONType): icd.JSONField => {
  const { key, name, pointer, units } = field;
  const base = { key, name, pointer, units, scale: 1, offset: 0 };
  if (isTimeFormat(type))
    return {
      ...base,
      type: "timestamp",
      timeFormat: type,
      // The default holds a time, not its encoding, so it outlives a format change.
      default: field.type === "timestamp" ? field.default : undefined,
    };
  if (isNumberKind(type)) {
    const { scale, offset, min, max } = field;
    const number = { ...base, scale, offset, min, max };
    const options = Task.Payload.readOptionsOf(
      NUMBER_DATA_TYPES[type],
      numberOptionsOf(field),
    );
    switch (options.type) {
      case "float":
        return {
          ...number,
          type: options.type,
          enumValues: options.enumValues,
          default: options.fill,
        };
      case "integer":
        return {
          ...number,
          type: options.type,
          enumValues: options.enumValues,
          default: options.fill,
        };
      case "unsigned":
        return {
          ...number,
          type: options.type,
          enumValues: options.enumValues,
          default: options.fill,
        };
      default:
        throw new Error(`${type} reads as ${options.type}`);
    }
  }
  if (type === "string" || type === "boolean") return { ...base, type, enumValues: [] };
  return { ...base, type };
};

/** The cell holds the field's key, and its dialog edits the field. */
const options: Column<icd.JSONField> = {
  name: "Options",
  render: (field, path, { preview, variant, "aria-label": ariaLabel }) => (
    <Options
      path={path}
      field={field}
      preview={preview}
      variant={variant}
      aria-label={ariaLabel}
    />
  ),
  width: 30,
  get: ({ key }) => key,
  set: (f) => f,
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
      get: ({ kind }) => kind,
      set: (f, cell) => ({ ...f, kind: cell as NumberKind }),
    },
    ...conversion<icd.BinaryField>(),
    multiplexor,
    multiplexValues,
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
  json: [
    name(),
    {
      name: "Pointer",
      render: "text",
      get: ({ pointer }) => pointer,
      set: (f, cell) => ({ ...f, pointer: String(cell) }),
    },
    {
      name: "Type",
      render: "jsonType",
      width: 34,
      get: jsonTypeOf,
      set: (f, cell) => withJSONType(f, cell as JSONType),
    },
    ...conversion<icd.JSONField>(),
    options,
  ],
};

const createJSON = (name: string): icd.JSONField => ({
  key: uuid.create(),
  name,
  scale: 1,
  offset: 0,
  units: "",
  type: "float",
  pointer: "",
  enumValues: [],
});

const createField = (encoding: Encoding, name: string = ""): Field => {
  if (encoding === "json") return createJSON(name);
  const base = { key: uuid.create(), name, scale: 1, offset: 0, units: "" };
  switch (encoding) {
    case "binary":
      return {
        ...base,
        startBit: 0,
        bitLength: 8,
        byteOrder: "little_endian",
        kind: "unsigned",
        multiplexValues: [],
      };
    case "delimited":
      return { ...base, encoding, position: 0 };
    case "tagged":
      return { ...base, encoding, tag: "" };
  }
};

const uniqueFieldName = (names: string[]): string => {
  const taken = new Set(names);
  let i = names.length + 1;
  while (taken.has(`Field ${i}`)) i++;
  return `Field ${i}`;
};

/** @returns a float field that reads the whole document, named apart from taken. */
export const createJSONField = (taken: string[]): icd.JSONField =>
  createJSON(uniqueFieldName(taken));

/**
 * Rebuilds the fields a grid shows from its rows. Each row updates the field at its
 * index and keeps that field's key, which other fields and tasks reference. Only
 * changed cells apply, so a cell that resets others keeps its effect.
 */
const fieldsFromRows = (
  encoding: Encoding,
  columns: Column<Field>[],
  prevFields: Field[],
  prevRows: Input.TableCell[][],
  rows: Input.TableCell[][],
): Field[] =>
  rows.map((row, i) =>
    columns.reduce<Field>(
      (field, column, j) =>
        row[j] === prevRows[i]?.[j] ? field : column.set(field, row[j]),
      prevFields[i] ?? createField(encoding),
    ),
  );

/** A change to one of these picks a new union member, so it replaces the field. */
const DISCRIMINATORS = ["encoding", "type"] as const;

/**
 * Writes each changed property of a field as its own edit, so another client's edit to
 * a different property of the field survives.
 */
const writeField = (
  { set, remove }: Pick<Form.ContextValue, "set" | "remove">,
  entry: deep.Path<unknown>,
  prev: Field,
  next: Field,
): void => {
  const before: record.Unknown = prev;
  const after: record.Unknown = next;
  if (DISCRIMINATORS.some((d) => before[d] !== after[d])) return set(entry, next);
  for (const prop of new Set([...Object.keys(before), ...Object.keys(after)])) {
    if (deep.equal(before[prop], after[prop])) continue;
    if (after[prop] === undefined) remove(entry[prop]);
    else set(entry[prop], after[prop]);
  }
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
  path: deep.Target<Field[]>,
): ReactElement<Input.TableColumnProps> => {
  const { name, render, width } = column;
  if (typeof render === "function")
    return (
      <Input.TableColumn key={index} name={name} width={width} type="string">
        {(props) => {
          const field = fields.find(({ key }) => key === props.value);
          if (field == null) throw new Error(`no field has the key ${props.value}`);
          return render(field, deep.widen(path)[field.key], props);
        }}
      </Input.TableColumn>
    );
  switch (render) {
    case "number":
      return <Input.TableColumn key={index} name={name} width={width} />;
    case "text":
      return <Input.TableColumn key={index} name={name} width={width} type="string" />;
    case "type":
      return selectColumn(column, index, "type", NUMBER_KIND_NAMES);
    case "jsonType":
      return selectColumn(column, index, "type", JSON_TYPE_NAMES);
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
  path: deep.Target<Field[]>,
  field: Field,
): boolean => key != null && deep.pathsMatch(key, `${String(path)}.${field.key}`);

export interface FieldsProps {
  /** The form path of the message's fields. */
  path: deep.Target<Field[]>;
  encoding: Encoding;
}

/** Edits the fields of one encoding in a message payload as a grid. */
export const Fields = ({ path, encoding }: FieldsProps): ReactElement => {
  const ctx = Form.useContext();
  const { push, remove } = Form.useFieldListUtils(path);
  const fields = Form.useFieldValue(path);
  const own = fields.filter((f) => encodingOf(f) === encoding);
  const status = Form.useStatuses(path).find(({ key }) =>
    own.some((f) => matchOwnField(key, path, f)),
  );
  const columns = COLUMNS[encoding] as Column<Field>[];
  const rows = own.map((f) => columns.map((c) => c.get(f)));
  const handleChange = (next: Input.TableCell[][]) => {
    const updated = fieldsFromRows(encoding, columns, own, rows, next);
    const entries = deep.widen<unknown>(path);
    own.forEach((field, i) => writeField(ctx, entries[field.key], field, updated[i]));
    if (updated.length > own.length) push(updated.slice(own.length));
  };
  const handleRemove = (index: number) => remove(own[index].key);
  // A paste adds several rows before any is stored. Each grid's first column is the
  // name.
  const createRow = (value: Input.TableCell[][]) => {
    const added = value.slice(own.length).map(([name]) => String(name));
    const fieldName = uniqueFieldName([...fields.map((f) => f.name), ...added]);
    return columns.map((c) => c.get(createField(encoding, fieldName)));
  };
  const failing = own.findIndex((f) => matchOwnField(status?.key, path, f));
  let helpText = status?.message;
  if (failing !== -1) helpText = `Field ${failing + 1}: ${helpText}`;
  return (
    <Flex.Box y gap="small" className={CSS.B("icd-fields")}>
      <Input.Table
        value={rows}
        onChange={handleChange}
        onRemove={handleRemove}
        createRow={createRow}
        resourceName="field"
        rowLabel={(i) => (i + 1).toString()}
        rowKey={(i) => own[i].key}
        preview={ctx.mode === "preview"}
      >
        {columns.map((column, i) => renderColumn(column, i, fields, path))}
      </Input.Table>
      {helpText != null && (
        <Input.HelpText variant={status?.variant}>{helpText}</Input.HelpText>
      )}
    </Flex.Box>
  );
};

export interface FieldsStatusProps {
  /** The form path of the message's fields. */
  path: deep.Target<unknown[]>;
}

/** Shows a problem with the fields of a message as a whole. */
export const FieldsStatus = ({ path }: FieldsStatusProps): ReactElement | null => {
  const status = Form.useStatuses(path).find(({ key }) => key === String(path));
  if (status == null) return null;
  return <Input.HelpText variant={status.variant}>{status.message}</Input.HelpText>;
};

const POINTER_INPUT_PROPS = { placeholder: "/temperature" } as const;

interface ScaleControlProps extends Input.Control<number> {
  offset: deep.Target<number>;
}

/** The scale, with the offset beside it in the same control. */
const ScaleControl = ({ offset, ...rest }: ScaleControlProps): ReactElement => (
  <div className={CSS.BE("icd-fields", "scale")}>
    <Input.Numeric {...rest} />
    <Form.NumericField path={offset} label="Offset" padHelpText={false} />
  </div>
);

export interface JSONFieldProps {
  /** The form path of the field. */
  path: deep.Path<icd.JSONField>;
  /** Actions in the header of the field's section. */
  actions?: ReactNode;
}

/**
 * Edits one field of a JSON message: its name, pointer, type, and conversion, then its
 * default and enum mapping when its kind has them.
 */
export const JSONField = ({ path, actions }: JSONFieldProps): ReactElement => {
  const { get, set, mode } = Form.useContext();
  const field = Form.useFieldValue(path);
  const type = jsonTypeOf(field);
  const handleType = (next: JSONType) => {
    if (next !== type) set(path, withJSONType(get(path).value, next));
  };
  return (
    <Form.Section title="Field" actions={actions}>
      <Form.TextField path={path.name} label="Name" padHelpText={false} />
      <Form.TextField
        path={path.pointer}
        label="Pointer"
        padHelpText={false}
        inputProps={POINTER_INPUT_PROPS}
      />
      <Input.Item label="Type" padHelpText={false}>
        <Select.Simple<JSONType>
          resourceName="type"
          value={type}
          onChange={handleType}
          variant="floating"
          allowNone={false}
          preview={mode === "preview"}
        >
          {Object.entries(JSON_TYPE_NAMES).map(([key, name]) => (
            <Select.Item key={key} itemKey={key}>
              {name}
            </Select.Item>
          ))}
        </Select.Simple>
      </Input.Item>
      {isNumberKind(type) && (
        <>
          <Form.Field<number> path={path.scale} label="Scale" padHelpText={false}>
            {(p) => <ScaleControl {...p} offset={path.offset} />}
          </Form.Field>
          <Form.TextField path={path.units} label="Units" padHelpText={false} />
        </>
      )}
      {showsDefault(field) && <Optional path={path} field={field} />}
      <EnumValues path={path} label="Enum mapping" />
    </Form.Section>
  );
};
