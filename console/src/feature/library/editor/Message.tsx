// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { library } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { TimeSpan } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Fields } from "@/feature/library/editor/Fields";
import { Identifier } from "@/feature/library/editor/Identifier";
import { OptionalField, parseInteger } from "@/feature/library/editor/OptionalField";

const FIELD_LABELS: Record<library.FieldType, string> = {
  binary: "Fields",
  delimited: "Delimited fields",
  tagged: "Tagged fields",
};

const FORMAT_ENCODINGS: Record<library.Format, library.FieldType[]> = {
  binary: ["binary"],
  text: ["delimited", "tagged"],
};

const formatPeriod = (period: TimeSpan): string => period.milliseconds.toString();

const parsePeriod = (text: string): TimeSpan | null => {
  const ms = Number(text.trim());
  return Number.isFinite(ms) && ms >= 0 ? TimeSpan.milliseconds(ms) : null;
};

const parseText = (text: string): string => text;

export interface MessageProps {
  /** The form path of the message entry. */
  path: string;
}

/** Edits a message: how frames match it, its layout, and its fields. */
export const Message = ({ path }: MessageProps): ReactElement => {
  const format = Form.useFieldValue<library.Format>(`${path}.format`);
  const fields = Form.useFieldValue<library.Field[]>(`${path}.fields`);
  const encodings = new Set(FORMAT_ENCODINGS[format]);
  fields.forEach(({ encoding }) => encodings.add(encoding));
  return (
    <Flex.Box y gap="medium">
      <Identifier path={path} />
      <Flex.Box x wrap gap="large">
        <Form.Field<library.Format> path={`${path}.format`} label="Format">
          {(p) => (
            <Select.Buttons<library.Format> {...p}>
              <Select.Item<library.Format> itemKey="binary">Binary</Select.Item>
              <Select.Item<library.Format> itemKey="text">Text</Select.Item>
            </Select.Buttons>
          )}
        </Form.Field>
        <OptionalField<number>
          path={`${path}.length`}
          label="Length (bytes)"
          placeholder="Variable"
          format={String}
          parse={parseInteger}
        />
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
        {format === "text" && (
          <Form.TextField path={`${path}.delimiter`} label="Delimiter" />
        )}
      </Flex.Box>
      {library.FIELD_TYPES.filter((encoding) => encodings.has(encoding)).map(
        (encoding) => (
          <Fields
            key={encoding}
            path={path}
            encoding={encoding}
            label={FIELD_LABELS[encoding]}
          />
        ),
      )}
    </Flex.Box>
  );
};
