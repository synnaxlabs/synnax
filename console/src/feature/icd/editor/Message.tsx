// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type icd } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Fields } from "@/feature/icd/editor/Fields";
import { FormattedField, parseInteger } from "@/feature/icd/editor/FormattedField";
import { Identifier, IdentifierFields } from "@/feature/icd/editor/Identifier";
import { useNestedStatus } from "@/feature/icd/editor/useNestedStatus";
import { CSS } from "@/platform/css";

const createPayload = (format: icd.PayloadType): icd.Payload => {
  switch (format) {
    case "binary":
      return { format, fields: [] };
    case "text":
      return { format, delimiter: ",", prefix: "", fields: [] };
  }
};

// The form store clones its values, which strips the TimeSpan class.
const formatPeriod = (period: CrudeTimeSpan): string =>
  new TimeSpan(period).milliseconds.toString();

const parsePeriod = (text: string): TimeSpan | null => {
  const ms = Number(text.trim());
  return Number.isFinite(ms) && ms >= 0 ? TimeSpan.milliseconds(ms) : null;
};

const parseText = (text: string): string => text;

interface FieldsStatusProps {
  /** The form path of the message payload. */
  path: string;
}

/** Shows a problem with the fields of the message as a whole. */
const FieldsStatus = ({ path }: FieldsStatusProps): ReactElement | null => {
  const fieldsPath = `${path}.fields`;
  const isFields = useCallback((key: string) => key === fieldsPath, [fieldsPath]);
  const status = useNestedStatus(fieldsPath, isFields);
  if (status == null) return null;
  return <Input.HelpText variant={status.variant}>{status.message}</Input.HelpText>;
};

export interface MessageProps {
  /** The form path of the message. */
  path: string;
}

/**
 * Edits a message: when it is sent, how frames match it, and its fields. A format
 * switch replaces the payload with an empty one of the new format.
 */
export const Message = ({ path }: MessageProps): ReactElement => {
  const payloadPath = `${path}.payload`;
  const { set } = Form.useContext();
  const format = Form.useFieldValue<icd.PayloadType>(`${payloadPath}.format`);
  const handleFormatChange = (next: icd.PayloadType) => {
    if (next !== format) set(payloadPath, createPayload(next));
  };
  return (
    <Form.Sections className={CSS.BE("icd-editor", "details")}>
      <Form.Section title="Message" className={CSS.BE("icd-editor", "pairs")}>
        <Form.TextField path={`${path}.name`} label="Name" />
        <Input.Item label="Format" padHelpText>
          <Select.Buttons<icd.PayloadType> value={format} onChange={handleFormatChange}>
            <Select.Item<icd.PayloadType> itemKey="binary">Binary</Select.Item>
            <Select.Item<icd.PayloadType> itemKey="text">Text</Select.Item>
          </Select.Buttons>
        </Input.Item>
        <FormattedField<CrudeTimeSpan>
          path={`${path}.period`}
          label="Send period"
          placeholder="On change"
          unit="ms"
          optional
          format={formatPeriod}
          parse={parsePeriod}
        />
        <FormattedField<string>
          path={`${path}.query`}
          label="Poll request"
          placeholder="None"
          optional
          format={String}
          parse={parseText}
        />
      </Form.Section>
      <Form.Section title="Frame" className={CSS.BE("icd-editor", "pairs")}>
        {format === "binary" ? (
          <>
            <Identifier path={payloadPath} />
            <FormattedField<number>
              path={`${payloadPath}.length`}
              label="Length"
              placeholder="Variable"
              unit="bytes"
              optional
              format={String}
              parse={parseInteger}
            />
            <IdentifierFields path={payloadPath} />
          </>
        ) : (
          <>
            <Form.TextField path={`${payloadPath}.prefix`} label="Prefix" />
            <Form.TextField path={`${payloadPath}.delimiter`} label="Delimiter" />
          </>
        )}
      </Form.Section>
      {format === "binary" ? (
        <Form.Section title="Fields">
          <Fields path={payloadPath} encoding="binary" />
          <FieldsStatus path={payloadPath} />
        </Form.Section>
      ) : (
        <>
          <Form.Section title="Delimited fields">
            <Fields path={payloadPath} encoding="delimited" />
          </Form.Section>
          <Form.Section title="Tagged fields">
            <Fields path={payloadPath} encoding="tagged" />
            <FieldsStatus path={payloadPath} />
          </Form.Section>
        </>
      )}
    </Form.Sections>
  );
};
