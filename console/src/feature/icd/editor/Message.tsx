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
import { type CrudeTimeSpan, deep, type payload, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { FormattedField, parseInteger } from "@/feature/icd/editor/FormattedField";
import { Identifier, IdentifierFields } from "@/feature/icd/editor/Identifier";
import { CSS } from "@/platform/css";
import { ICD } from "@/platform/icd";

const createPayload = (format: payload.Format): icd.Payload => {
  switch (format) {
    case "binary":
      return { format, fields: [] };
    case "text":
      return { format, delimiter: ",", prefix: "", fields: [] };
    case "json":
      return { format, fields: [] };
  }
};

const checkCycleTime = (span: TimeSpan): string | null => {
  if (span.lessThan(TimeSpan.NANOSECOND)) return "Cycle time must be 1ns or more";
  if (span.greaterThan(TimeSpan.MAX))
    return `Cycle time must be ${Input.formatTimeSpan(TimeSpan.MAX)} or less`;
  return null;
};

interface CycleTimeProps {
  path: deep.Target<CrudeTimeSpan | undefined>;
}

/** Edits the cycle time of a message. A number typed alone is in milliseconds. */
const CycleTime = ({ path }: CycleTimeProps): ReactElement => {
  const { set, mode } = Form.useContext();
  const state = Form.useFieldState(path, { optional: true });
  // The form store clones its values, which strips the TimeSpan class.
  const value = state?.value == null ? undefined : new TimeSpan(state.value);
  return (
    <Input.Item
      label="Cycle time"
      padHelpText
      helpText={state?.status.message}
      status={state?.status.variant}
    >
      <Input.TimeSpan
        inline
        unit={TimeSpan.MILLISECOND}
        precision={TimeSpan.NANOSECOND}
        value={value == null ? undefined : Number(value.valueOf())}
        onChange={(ns) => set(path, new TimeSpan(ns))}
        onClear={() => set(path, undefined)}
        check={checkCycleTime}
        placeholder="None"
        aria-label="Cycle time"
        preview={mode === "preview"}
      />
    </Input.Item>
  );
};

export interface MessageProps {
  /** The form path of the message. */
  path: deep.Path<icd.Message>;
}

/**
 * Edits a message: its format, its cycle time, how frames match it, and its fields. A
 * format switch replaces the payload with an empty one of the new format.
 */
export const Message = ({ path }: MessageProps): ReactElement => {
  const payloadPath = path.payload;
  const binary = deep.variant(payloadPath, "format", "binary");
  const text = deep.variant(payloadPath, "format", "text");
  const json = deep.variant(payloadPath, "format", "json");
  const { set, mode } = Form.useContext();
  const format = Form.useFieldValue(payloadPath.format);
  const handleFormatChange = (next: payload.Format) => {
    if (next !== format) set(payloadPath, createPayload(next));
  };
  return (
    <Form.Sections className={CSS.BE("icd-editor", "details")}>
      <Form.Section title="Message">
        <Flex.Box x>
          <Form.TextField path={path.name} label="Name" />
          <Input.Item label="Format" padHelpText>
            <Select.Buttons<payload.Format>
              value={format}
              onChange={handleFormatChange}
              preview={mode === "preview"}
            >
              <Select.Item<payload.Format> itemKey="binary">Binary</Select.Item>
              <Select.Item<payload.Format> itemKey="text">Text</Select.Item>
              <Select.Item<payload.Format> itemKey="json">JSON</Select.Item>
            </Select.Buttons>
          </Input.Item>
        </Flex.Box>
        <CycleTime path={path.cycleTime} />
      </Form.Section>
      {format === "binary" && (
        <>
          <Form.Section title="Frame">
            <Flex.Box x>
              <Identifier path={binary} />
              <FormattedField<number>
                path={binary.length}
                label="Length"
                placeholder="Variable"
                unit="bytes"
                optional
                format={String}
                parse={parseInteger}
              />
            </Flex.Box>
            <IdentifierFields path={binary} />
          </Form.Section>
          <Form.Section title="Fields">
            <ICD.Fields path={binary.fields} encoding="binary" />
            <ICD.FieldsStatus path={binary.fields} />
          </Form.Section>
        </>
      )}
      {format === "text" && (
        <>
          <Form.Section title="Frame">
            <Flex.Box x>
              <Form.TextField path={text.prefix} label="Prefix" />
              <Form.TextField path={text.delimiter} label="Delimiter" />
            </Flex.Box>
          </Form.Section>
          <Form.Section title="Delimited fields">
            <ICD.Fields path={text.fields} encoding="delimited" />
          </Form.Section>
          <Form.Section title="Tagged fields">
            <ICD.Fields path={text.fields} encoding="tagged" />
            <ICD.FieldsStatus path={text.fields} />
          </Form.Section>
        </>
      )}
      {format === "json" && (
        <>
          <Form.Section title="Documents">
            <ICD.Match path={json} />
          </Form.Section>
          <Form.Section title="Fields">
            <ICD.Fields path={json.fields} encoding="json" />
            <ICD.FieldsStatus path={json.fields} />
          </Form.Section>
        </>
      )}
    </Form.Sections>
  );
};
