// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { type CrudeDataType, deep, type payload } from "@synnaxlabs/x";
import { type ReactElement, useEffect } from "react";

import { Form as PlatformForm } from "@/platform/form";
import { EnumMappingSection } from "@/platform/task/payload/EnumMappingSection";
import { channelKinds, kindName } from "@/platform/task/payload/kinds";
import { TimeFormatField } from "@/platform/task/payload/TimeFormatField";

/** The kind and the options of a field that writes the samples of a channel. */
export type ChannelField =
  | { type: "float"; enumValues: payload.FloatEntry[] }
  | { type: "integer"; enumValues: payload.IntegerEntry[] }
  | { type: "unsigned"; enumValues: payload.UnsignedEntry[] }
  | { type: "string"; enumValues: payload.StringEntry[] }
  | { type: "boolean"; enumValues: payload.BooleanEntry[] }
  | { type: "timestamp"; timeFormat: payload.TimeFormat }
  | { type: "json" }
  | { type: "uuid" }
  | { type: "bytes" };

type NumericKind = "float" | "integer" | "unsigned";

const isNumericKind = (kind: payload.Kind): kind is NumericKind =>
  kind === "float" || kind === "integer" || kind === "unsigned";

interface NumericEntry {
  label: string;
  value: number | string;
}

const numericEntries = (field: ChannelField): NumericEntry[] =>
  field.type === "float" || field.type === "integer" || field.type === "unsigned"
    ? field.enumValues
    : [];

const MAX_EXACT_FLOAT = 2n ** 53n;

const RANGES: Record<"integer" | "unsigned", [bigint, bigint]> = {
  integer: [-(2n ** 63n), 2n ** 63n - 1n],
  unsigned: [0n, 2n ** 64n - 1n],
};

const exactInteger = (value: number | string): bigint | null => {
  if (typeof value === "number") return Number.isInteger(value) ? BigInt(value) : null;
  return /^-?[0-9]+$/.test(value) ? BigInt(value) : null;
};

/** Returns value as an entry value of kind, or null when it does not stay exact. */
const convertValue = (
  value: number | string,
  kind: NumericKind,
): number | string | null => {
  if (kind === "float" && typeof value === "number") return value;
  const n = exactInteger(value);
  if (n == null) return null;
  if (kind === "float")
    return n >= -MAX_EXACT_FLOAT && n <= MAX_EXACT_FLOAT ? Number(n) : null;
  const [min, max] = RANGES[kind];
  return n >= min && n <= max ? n.toString() : null;
};

/** The options of a field written as another kind. */
export interface KindOptions {
  timeFormat?: payload.TimeFormat;
  enumValues?: NumericEntry[];
}

/**
 * Returns the options of field written as kind: ISO 8601 for a timestamp, and, between
 * numeric kinds, each entry that stays exact.
 */
export const kindOptions = (field: ChannelField, kind: payload.Kind): KindOptions => {
  if (kind === "timestamp") return { timeFormat: "iso8601" };
  if (!isNumericKind(kind)) return {};
  return {
    enumValues: numericEntries(field).flatMap(({ label, value }) => {
      const converted = convertValue(value, kind);
      return converted == null ? [] : [{ label, value: converted }];
    }),
  };
};

export interface KindFieldsProps<F extends ChannelField> {
  path: deep.Target<F>;
  /** The data type of the channel, or undefined while it is not known. */
  dataType: CrudeDataType | undefined;
  /** Returns field written as kind, with the defaults of kind. */
  asKind: (field: F, kind: payload.Kind) => F;
}

/**
 * Selects the kind a field writes the samples of its channel as, and the time format of
 * a timestamp. A kind the data type does not allow becomes the data type's own kind.
 */
export const KindFields = <F extends ChannelField>({
  path,
  dataType,
  asKind,
}: KindFieldsProps<F>): ReactElement => {
  const field = Form.useFieldValue(path);
  const { set } = Form.useContext();
  const kinds = dataType == null ? null : channelKinds(dataType);
  const allowed = kinds == null || kinds.includes(field.type);
  useEffect(() => {
    if (kinds != null && !allowed) set(path, asKind(field, kinds[0]));
  }, [kinds?.[0], allowed]);
  return (
    <>
      {kinds != null && kinds.length > 1 && (
        <Form.Field<F> path={path} label="Output type" padHelpText={false}>
          {({ value, onChange }) => (
            <Select.Simple<payload.Kind>
              value={value.type}
              onChange={(kind: payload.Kind) => onChange(asKind(value, kind))}
              resourceName="kind"
            >
              {kinds.map((kind) => (
                <Select.Item key={kind} itemKey={kind}>
                  {kindName(kind)}
                </Select.Item>
              ))}
            </Select.Simple>
          )}
        </Form.Field>
      )}
      {field.type === "timestamp" && (
        <TimeFormatField
          path={
            deep.variant(deep.widen<ChannelField>(path), "type", "timestamp").timeFormat
          }
          label="Time format"
        />
      )}
    </>
  );
};

export interface EntriesSectionProps {
  path: deep.Target<ChannelField>;
}

/** Maps the channel values of a field to the labels it writes in their place. */
export const EntriesSection = ({ path }: EntriesSectionProps): ReactElement | null => {
  const type = Form.useFieldValue(deep.widen(path).type);
  const field = deep.widen(path);
  switch (type) {
    case "float":
      return (
        <EnumMappingSection
          path={deep.variant(field, "type", "float").enumValues}
          valueFirst
        />
      );
    case "integer":
    case "unsigned":
    case "string":
      return (
        <Form.Section title="Enum mapping">
          <PlatformForm.KeyValueEditor
            path={deep.variant(field, "type", type).enumValues}
            resourceName="mapping"
            keyField="label"
            keyPlaceholder="String (e.g. ON)"
            valuePlaceholder={type === "string" ? "Text" : "Integer"}
            valueFirst
          />
        </Form.Section>
      );
    case "boolean":
      return (
        <Form.Section title="Enum mapping">
          <PlatformForm.KeyValueEditor
            path={deep.variant(field, "type", "boolean").enumValues}
            resourceName="mapping"
            keyField="label"
            keyPlaceholder="String (e.g. ON)"
            valueType="boolean"
            valueFirst
          />
        </Form.Section>
      );
    default:
      return null;
  }
};
