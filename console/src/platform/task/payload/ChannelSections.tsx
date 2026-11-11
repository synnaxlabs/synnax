// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { Channel } from "@synnaxlabs/pluto";
import { DataType, deep, type payload } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";

import { type Form as PlatformForm } from "@/platform/form";
import { DataTypeField } from "@/platform/task/payload/DataTypeField";
import { EnumMappingSection } from "@/platform/task/payload/EnumMappingSection";
import { JSON_TYPE_ITEMS } from "@/platform/task/payload/JSONTypeItems";
import { TimeFormatField } from "@/platform/task/payload/TimeFormatField";

const renderSelectJSONType = Component.renderProp(
  (
    p: Omit<Select.SingleSimpleProps<payload.JSONType>, "children" | "resourceName">,
  ) => (
    <Select.Simple<payload.JSONType> {...p} resourceName="JSON type">
      {JSON_TYPE_ITEMS}
    </Select.Simple>
  ),
);

interface CommandField {
  channel: channel.Key;
  dataType: string;
  pointer: string;
  jsonType: payload.JSONType;
  timeFormat?: payload.TimeFormat;
  enumValues: PlatformForm.Entry<"label", number>[];
}

export interface ChannelSectionsProps {
  /** The path of the command channel field of a write target. */
  path: deep.Target<CommandField>;
  /** Says what an empty pointer sends. */
  pointerPlaceholder: string;
  /** Leads the channel section, such as a field that names the channel. */
  children?: ReactNode;
}

/**
 * Edits how a write target places the value of its command channel in the payload. A
 * timestamp channel adds a time format, and a string JSON type adds an enum mapping.
 */
export const ChannelSections = ({
  path,
  pointerPlaceholder,
  children,
}: ChannelSectionsProps): ReactElement => {
  const field = deep.widen(path);
  const key = Form.useFieldValue(field.channel);
  const jsonType = Form.useFieldValue(field.jsonType);
  const query = key === 0 ? null : { key };
  const { data: dataType } = Channel.useResultDataType(query);
  const { set } = Form.useContext();
  // The Driver rejects enum values on any JSON type but string.
  const handleJSONTypeChange = (value: payload.JSONType) => {
    if (value !== "string") set(field.enumValues, []);
  };
  const pointerInputProps = { placeholder: pointerPlaceholder };
  return (
    <>
      <Form.Section title="Channel">
        {children}
        <DataTypeField path={field.dataType} channel={key} hideVariableDensity />
        <Form.TextField
          path={field.pointer}
          label="Pointer"
          padHelpText={false}
          inputProps={pointerInputProps}
        />
        <Form.Field
          path={field.jsonType}
          label="JSON type"
          padHelpText={false}
          onChange={handleJSONTypeChange}
        >
          {renderSelectJSONType}
        </Form.Field>
        {dataType != null && DataType.TIMESTAMP.equals(dataType) && (
          <TimeFormatField path={field.timeFormat} label="Time format" />
        )}
      </Form.Section>
      {jsonType === "string" && (
        <EnumMappingSection path={field.enumValues} valueFirst />
      )}
    </>
  );
};
