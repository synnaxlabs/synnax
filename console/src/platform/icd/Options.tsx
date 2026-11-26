// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/icd/Options.css";

import type { icd } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { deep, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { CSS } from "@/platform/css";
import { EnumValues } from "@/platform/icd/EnumValues";
import { ValueField } from "@/platform/icd/ValueField";

/** A JSON field of a kind with a default and an enum mapping. */
type Mappable =
  | icd.FloatJSONField
  | icd.IntegerJSONField
  | icd.UnsignedJSONField
  | icd.StringJSONField
  | icd.BooleanJSONField;

type Type = icd.JSONField["type"];

const MAPPABLE: Record<Mappable["type"], true> = {
  float: true,
  integer: true,
  unsigned: true,
  string: true,
  boolean: true,
};

const isMappable = (field: icd.JSONField): field is Mappable => field.type in MAPPABLE;

/** A JSON field of a kind with a min and a max. */
type Limited = icd.FloatJSONField | icd.IntegerJSONField | icd.UnsignedJSONField;

const LIMITED: Record<Limited["type"], true> = {
  float: true,
  integer: true,
  unsigned: true,
};

const isLimited = (field: icd.JSONField): field is Limited => field.type in LIMITED;

const ZERO_DEFAULTS: {
  [T in Type]: NonNullable<Extract<icd.JSONField, { type: T }>["default"]>;
} = {
  float: 0,
  integer: "0",
  unsigned: "0",
  string: "",
  boolean: false,
  timestamp: { type: "now" },
  uuid: { type: "random" },
  json: "null",
  bytes: "",
};

const TIMESTAMP_DEFAULTS: Record<icd.TimestampDefaultType, () => icd.TimestampDefault> =
  {
    now: () => ({ type: "now" }),
    constant: () => ({
      type: "constant",
      value: TimeStamp.now().truncate(TimeSpan.SECOND),
    }),
  };

const UUID_DEFAULTS: Record<icd.UUIDDefaultType, () => icd.UUIDDefault> = {
  random: () => ({ type: "random" }),
  constant: () => ({ type: "constant", value: uuid.create() }),
};

/** @returns the default of the field as text, or null when it has none. */
export const formatDefault = (field: icd.JSONField): string | null => {
  if (field.default == null) return null;
  switch (field.type) {
    case "string":
    case "bytes":
      return JSON.stringify(field.default);
    case "boolean":
      return field.default ? "True" : "False";
    case "timestamp":
      return field.default.type === "now"
        ? "Now"
        : field.default.value.toString("dateTime", "local");
    case "uuid":
      return field.default.type === "random" ? "Random" : field.default.value;
    default:
      return String(field.default);
  }
};

/** @returns what the field's options set, or null when they set nothing. */
const summarize = (field: icd.JSONField): string | null => {
  const parts: string[] = [];
  const shown = formatDefault(field);
  if (shown != null) parts.push(`Default ${shown}`);
  if (field.min != null) parts.push(`Min ${field.min}`);
  if (field.max != null) parts.push(`Max ${field.max}`);
  const count = isMappable(field) ? field.enumValues.length : 0;
  if (count > 0) parts.push(`${count} ${count === 1 ? "mapping" : "mappings"}`);
  return parts.length > 0 ? parts.join(", ") : null;
};

const OPTIONAL_INPUT_PROPS = {
  tooltip: "Write the default when the document has no value.",
} as const;

interface DefaultProps {
  path: deep.Path<icd.JSONField>;
  type: Type;
}

interface TimestampDefaultProps {
  path: deep.Path<icd.TimestampDefault>;
}

/** Edits whether a timestamp default is now, and its time when it is constant. */
const TimestampDefault = ({ path }: TimestampDefaultProps): ReactElement => {
  const { type } = Form.useFieldValue(path);
  return (
    <>
      <Form.Field<icd.TimestampDefault> path={path} label="Default" padHelpText={false}>
        {({ value, onChange, preview }) => (
          <Select.Buttons<icd.TimestampDefaultType>
            value={value.type}
            onChange={(next: icd.TimestampDefaultType) =>
              onChange(TIMESTAMP_DEFAULTS[next]())
            }
            preview={preview}
          >
            <Select.Item<icd.TimestampDefaultType> itemKey="now">Now</Select.Item>
            <Select.Item<icd.TimestampDefaultType> itemKey="constant">
              Constant
            </Select.Item>
          </Select.Buttons>
        )}
      </Form.Field>
      {type === "constant" && (
        <ValueField type="timestamp" path={deep.variant(path, "type", type).value} />
      )}
    </>
  );
};

interface UUIDDefaultProps {
  path: deep.Path<icd.UUIDDefault>;
}

/** Edits whether a UUID default is random, and its UUID when it is constant. */
const UUIDDefault = ({ path }: UUIDDefaultProps): ReactElement => {
  const { type } = Form.useFieldValue(path);
  return (
    <>
      <Form.Field<icd.UUIDDefault> path={path} label="Default" padHelpText={false}>
        {({ value, onChange, preview }) => (
          <Select.Buttons<icd.UUIDDefaultType>
            value={value.type}
            onChange={(next: icd.UUIDDefaultType) => onChange(UUID_DEFAULTS[next]())}
            preview={preview}
          >
            <Select.Item<icd.UUIDDefaultType> itemKey="random">Random</Select.Item>
            <Select.Item<icd.UUIDDefaultType> itemKey="constant">Constant</Select.Item>
          </Select.Buttons>
        )}
      </Form.Field>
      {type === "constant" && (
        <ValueField type="uuid" path={deep.variant(path, "type", type).value} />
      )}
    </>
  );
};

/** Edits the default of the field's kind. */
const Default = ({ path, type }: DefaultProps): ReactElement => {
  if (type === "timestamp")
    return (
      <TimestampDefault path={deep.present(deep.variant(path, "type", type).default)} />
    );
  if (type === "uuid")
    return (
      <UUIDDefault path={deep.present(deep.variant(path, "type", type).default)} />
    );
  return (
    <ValueField
      type={type}
      path={deep.variant(path, "type", type).default}
      label="Default"
    />
  );
};

export interface OptionalProps {
  path: deep.Path<icd.JSONField>;
  field: icd.JSONField;
}

/** Edits whether a field is optional, and its default when it is. */
export const Optional = ({ path, field }: OptionalProps): ReactElement => {
  const { set, remove, mode } = Form.useContext();
  const defaultPath = deep.variant(path, "type", field.type).default;
  const handleOptional = (optional: boolean) => {
    if (optional) set(defaultPath, ZERO_DEFAULTS[field.type]);
    else remove(defaultPath);
  };
  return (
    <>
      <Input.Item label="Optional" padHelpText={false}>
        <Input.Checkbox
          value={field.default != null}
          onChange={handleOptional}
          preview={mode === "preview"}
          {...OPTIONAL_INPUT_PROPS}
        />
      </Input.Item>
      {field.default != null && <Default path={path} type={field.type} />}
    </>
  );
};

interface LimitProps {
  path: deep.Target<number | undefined>;
  label: string;
}

/** Edits a min or a max. Clearing it removes the limit. */
const Limit = ({ path, label }: LimitProps): ReactElement => {
  const { set, remove, mode } = Form.useContext();
  const state = Form.useFieldState(path, { optional: true });
  return (
    <Input.Item
      label={label}
      padHelpText={false}
      helpText={state?.status.message}
      status={state?.status.variant}
    >
      <Input.Numeric
        value={state?.value}
        onChange={(v) => set(path, v)}
        onClear={() => remove(path)}
        placeholder="None"
        aria-label={label}
        preview={mode === "preview"}
      />
    </Input.Item>
  );
};

/** Edits the min and max of a number field: the values that a write task accepts. */
const Limits = ({ path, field }: OptionalProps): ReactElement | null =>
  isLimited(field) ? (
    <>
      <Limit path={path.min} label="Min" />
      <Limit path={path.max} label="Max" />
    </>
  ) : null;

const OptionsForm = ({ path, field }: OptionalProps): ReactElement => (
  <Form.Sections>
    <Form.Section>
      <Optional path={path} field={field} />
      <Limits path={path} field={field} />
      {isMappable(field) && <EnumValues path={path} label="Enum mapping" />}
    </Form.Section>
  </Form.Sections>
);

export interface OptionsProps extends Pick<
  Input.TableCellProps,
  "preview" | "variant" | "aria-label"
> {
  /** The form path of the field. */
  path: deep.Path<icd.JSONField>;
  field: icd.JSONField;
}

/**
 * A grid cell that sums up the default, the min and max, and the enum mapping of a JSON
 * field, and opens a dialog that edits them. A read-only field that sets none of them
 * shows "None" and opens nothing.
 */
export const Options = ({
  path,
  field,
  preview,
  variant,
  "aria-label": ariaLabel,
}: OptionsProps): ReactElement => {
  const summary = summarize(field);
  const opens = preview !== true || summary != null;
  return (
    <Dialog.Frame variant="floating">
      <Dialog.Trigger
        variant={variant}
        preview={!opens}
        aria-label={ariaLabel}
        className={CSS.BE("icd-options", "trigger")}
      >
        <Text.Text
          el="span"
          overflow="fade"
          className={CSS.cls(
            CSS.BE("icd-options", "summary"),
            summary == null && CSS.BEM("icd-options", "summary", "empty"),
          )}
        >
          {summary ?? "None"}
        </Text.Text>
      </Dialog.Trigger>
      <Dialog.Dialog className={CSS.BE("icd-options", "dialog")}>
        <OptionsForm path={path} field={field} />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
