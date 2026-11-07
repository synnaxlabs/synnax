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
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Fields } from "@/feature/icd/editor/Fields";
import { FormattedField, parseInteger } from "@/feature/icd/editor/FormattedField";
import { Identifier, IdentifierFields } from "@/feature/icd/editor/Identifier";
import { CSS } from "@/platform/css";

const createPayload = (format: icd.PayloadType): icd.Payload => {
  switch (format) {
    case "binary":
      return { format, fields: [] };
    case "text":
      return { format, delimiter: ",", prefix: "", fields: [] };
    case "json":
      return { format, fields: [] };
  }
};

interface FieldsStatusProps {
  /** The form path of the message's fields. */
  path: deep.Target<unknown[]>;
}

/** Shows a problem with the fields of the message as a whole. */
const FieldsStatus = ({ path }: FieldsStatusProps): ReactElement | null => {
  const status = Form.useStatuses(path).find(({ key }) => key === String(path));
  if (status == null) return null;
  return <Input.HelpText variant={status.variant}>{status.message}</Input.HelpText>;
};

export interface MessageProps {
  /** The form path of the message. */
  path: deep.Path<icd.Message>;
}

/**
 * Edits a message: its format, how frames match it, and its fields. A format switch
 * replaces the payload with an empty one of the new format.
 */
export const Message = ({ path }: MessageProps): ReactElement => {
  const payloadPath = path.payload;
  const binary = deep.variant(payloadPath, "format", "binary");
  const text = deep.variant(payloadPath, "format", "text");
  const json = deep.variant(payloadPath, "format", "json");
  const { set, mode } = Form.useContext();
  const format = Form.useFieldValue(payloadPath.format);
  const handleFormatChange = (next: icd.PayloadType) => {
    if (next !== format) set(payloadPath, createPayload(next));
  };
  return (
    <Form.Sections className={CSS.BE("icd-editor", "details")}>
      <Form.Section title="Message">
        <Flex.Box x>
          <Form.TextField path={path.name} label="Name" />
          <Input.Item label="Format" padHelpText>
            <Select.Buttons<icd.PayloadType>
              value={format}
              onChange={handleFormatChange}
              preview={mode === "preview"}
            >
              <Select.Item<icd.PayloadType> itemKey="binary">Binary</Select.Item>
              <Select.Item<icd.PayloadType> itemKey="text">Text</Select.Item>
              <Select.Item<icd.PayloadType> itemKey="json">JSON</Select.Item>
            </Select.Buttons>
          </Input.Item>
        </Flex.Box>
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
            <Fields path={binary.fields} encoding="binary" />
            <FieldsStatus path={binary.fields} />
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
            <Fields path={text.fields} encoding="delimited" />
          </Form.Section>
          <Form.Section title="Tagged fields">
            <Fields path={text.fields} encoding="tagged" />
            <FieldsStatus path={text.fields} />
          </Form.Section>
        </>
      )}
      {format === "json" && (
        <Form.Section title="Fields">
          <Fields path={json.fields} encoding="json" />
          <FieldsStatus path={json.fields} />
        </Form.Section>
      )}
    </Form.Sections>
  );
};
