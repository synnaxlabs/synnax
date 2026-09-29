// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type library } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Fields } from "@/feature/library/editor/Fields";
import { Identifier } from "@/feature/library/editor/Identifier";
import { OptionalField, parseInteger } from "@/feature/library/editor/OptionalField";

const createPayload = (format: library.PayloadType): library.Payload => {
  switch (format) {
    case "binary":
      return { format, fields: [] };
    case "text":
      return { format, delimiter: ",", prefix: "", fields: [] };
  }
};

const formatPeriod = (period: TimeSpan): string => period.milliseconds.toString();

const parsePeriod = (text: string): TimeSpan | null => {
  const ms = Number(text.trim());
  return Number.isFinite(ms) && ms >= 0 ? TimeSpan.milliseconds(ms) : null;
};

const parseText = (text: string): string => text;

interface PayloadProps {
  /** The form path of the payload. */
  path: string;
}

const Binary = ({ path }: PayloadProps): ReactElement => (
  <>
    <Flex.Box x wrap gap="large">
      <Identifier path={path} />
      <OptionalField<number>
        path={`${path}.length`}
        label="Length (bytes)"
        placeholder="Variable"
        format={String}
        parse={parseInteger}
      />
    </Flex.Box>
    <Fields path={path} encoding="binary" label="Fields" />
  </>
);

const Text = ({ path }: PayloadProps): ReactElement => (
  <>
    <Flex.Box x wrap gap="large">
      <Form.TextField path={`${path}.prefix`} label="Prefix" />
      <Form.TextField path={`${path}.delimiter`} label="Delimiter" />
    </Flex.Box>
    <Fields path={path} encoding="delimited" label="Delimited fields" />
    <Fields path={path} encoding="tagged" label="Tagged fields" />
  </>
);

export interface MessageProps {
  /** The form path of the message entry. */
  path: string;
}

/**
 * Edits a message: its payload, how frames match it, and when it is sent. A format
 * switch replaces the payload with an empty one of the new format.
 */
export const Message = ({ path }: MessageProps): ReactElement => {
  const payloadPath = `${path}.payload`;
  const { set } = Form.useContext();
  const format = Form.useFieldValue<library.PayloadType>(`${payloadPath}.format`);
  const handleFormatChange = (next: library.PayloadType) => {
    if (next !== format) set(payloadPath, createPayload(next));
  };
  return (
    <Flex.Box y gap="medium">
      <Flex.Box x wrap gap="large">
        <Input.Item label="Format" padHelpText>
          <Select.Buttons<library.PayloadType>
            value={format}
            onChange={handleFormatChange}
          >
            <Select.Item<library.PayloadType> itemKey="binary">Binary</Select.Item>
            <Select.Item<library.PayloadType> itemKey="text">Text</Select.Item>
          </Select.Buttons>
        </Input.Item>
        <OptionalField<TimeSpan>
          path={`${path}.period`}
          label="Period (ms)"
          placeholder="On change"
          format={formatPeriod}
          parse={parsePeriod}
        />
        <OptionalField<string>
          path={`${path}.query`}
          label="Query"
          placeholder="None"
          format={String}
          parse={parseText}
        />
      </Flex.Box>
      {format === "binary" ? (
        <Binary path={payloadPath} />
      ) : (
        <Text path={payloadPath} />
      )}
    </Flex.Box>
  );
};
