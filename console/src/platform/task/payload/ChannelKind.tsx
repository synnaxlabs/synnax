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
        <Form.Field<F> path={path} label="Write as" padHelpText={false}>
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
