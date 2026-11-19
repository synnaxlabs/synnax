// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/icd/Options.css";

import { type icd } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { deep, type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { EnumValues } from "@/platform/icd/EnumValues";

/** A JSON field of a kind with a default and an enum mapping. */
type Mappable =
  | icd.FloatJSONField
  | icd.IntegerJSONField
  | icd.UnsignedJSONField
  | icd.StringJSONField
  | icd.BooleanJSONField;

type Kind = Mappable["type"];

const ZERO_DEFAULTS: {
  [K in Kind]: NonNullable<Extract<Mappable, { type: K }>["default"]>;
} = {
  float: 0,
  integer: "0",
  unsigned: "0",
  string: "",
  boolean: false,
};

const isMappable = (field: icd.JSONField): field is Mappable =>
  field.type in ZERO_DEFAULTS;

/** A JSON field with a default that the Console shows. */
type Defaulted = Mappable | icd.TimestampJSONField;

/**
 * @returns whether the Console shows the default of the field. A timestamp field shows
 * one only when it has one, which the Console can remove but not set.
 */
export const showsDefault = (field: icd.JSONField): field is Defaulted =>
  isMappable(field) || (field.type === "timestamp" && field.default != null);

const formatDefault = (field: Defaulted): string | null => {
  if (field.default == null) return null;
  switch (field.type) {
    case "string":
      return JSON.stringify(field.default);
    case "boolean":
      return field.default ? "True" : "False";
    case "timestamp":
      return field.default.toString("dateTime", "local");
    default:
      return String(field.default);
  }
};

/** @returns what the field's options set, or null when they set nothing. */
const summarize = (field: icd.JSONField): string | null => {
  if (!showsDefault(field)) return null;
  const parts: string[] = [];
  const shown = formatDefault(field);
  if (shown != null) parts.push(`Default ${shown}`);
  const count = isMappable(field) ? field.enumValues.length : 0;
  if (count > 0) parts.push(`${count} ${count === 1 ? "mapping" : "mappings"}`);
  return parts.length > 0 ? parts.join(", ") : null;
};

const OPTIONAL_INPUT_PROPS = {
  tooltip: "Write the default when the document has no value.",
} as const;

const BOOLEANS = { false: false, true: true } as const;

type BooleanKey = keyof typeof BOOLEANS;

interface DefaultProps {
  path: deep.Path<icd.JSONField>;
  type: Defaulted["type"];
}

/** Edits the default of the field's kind. */
const Default = ({ path, type }: DefaultProps): ReactElement => {
  switch (type) {
    case "float":
      return (
        <Form.NumericField
          path={deep.variant(path, "type", type).default}
          label="Default"
          padHelpText={false}
          emptiable={false}
        />
      );
    case "integer":
    case "unsigned":
    case "string":
      return (
        <Form.TextField
          path={deep.variant(path, "type", type).default}
          label="Default"
          padHelpText={false}
          emptiable={false}
        />
      );
    case "boolean":
      return (
        <Form.Field<boolean>
          path={deep.present(deep.variant(path, "type", type).default)}
          label="Default"
          padHelpText={false}
        >
          {({ value, onChange, preview }) => (
            <Select.Buttons<BooleanKey>
              value={String(value) as BooleanKey}
              onChange={(key: BooleanKey) => onChange(BOOLEANS[key])}
              preview={preview}
            >
              <Select.Item<BooleanKey> itemKey="false">False</Select.Item>
              <Select.Item<BooleanKey> itemKey="true">True</Select.Item>
            </Select.Buttons>
          )}
        </Form.Field>
      );
    case "timestamp":
      return (
        <Form.Field<TimeStamp>
          path={deep.present(deep.variant(path, "type", type).default)}
          label="Default"
          padHelpText={false}
        >
          {({ value }) => (
            <Text.Text>{value.toString("preciseDate", "local")}</Text.Text>
          )}
        </Form.Field>
      );
  }
};

export interface OptionalProps {
  path: deep.Path<icd.JSONField>;
  field: Defaulted;
}

/** Edits whether a field is optional, and its default when it is. */
export const Optional = ({ path, field }: OptionalProps): ReactElement => {
  const { set, remove, mode } = Form.useContext();
  const defaultPath = deep.variant(path, "type", field.type).default;
  const handleOptional = (optional: boolean) => {
    if (!optional) remove(defaultPath);
    else if (isMappable(field)) set(defaultPath, ZERO_DEFAULTS[field.type]);
    else throw new Error(`The default of ${field.name} cannot be set`);
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

const OptionsForm = ({ path, field }: OptionalProps): ReactElement => (
  <Form.Sections>
    <Form.Section>
      <Flex.Box x>
        <Optional path={path} field={field} />
      </Flex.Box>
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
 * A grid cell that sums up the default and the enum mapping of a JSON field, and opens
 * a dialog that edits them. A field with no options shows "None" and opens nothing, as
 * does a read-only field that sets neither.
 */
export const Options = ({
  path,
  field,
  preview,
  variant,
  "aria-label": ariaLabel,
}: OptionsProps): ReactElement => {
  const summary = summarize(field);
  const opens = showsDefault(field) && (preview !== true || summary != null);
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
      {showsDefault(field) && (
        <Dialog.Dialog className={CSS.BE("icd-options", "dialog")}>
          <OptionsForm path={path} field={field} />
        </Dialog.Dialog>
      )}
    </Dialog.Frame>
  );
};
