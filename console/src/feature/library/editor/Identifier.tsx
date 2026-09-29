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
import { type ReactElement } from "react";

import { OptionalField, parseInteger } from "@/feature/library/editor/OptionalField";

type Type = library.IdentifierType | "none";

const NAMES: Record<Type, string> = {
  none: "None",
  can: "CAN",
  arinc429: "ARINC 429",
  mil1553: "MIL-STD-1553",
  field: "Field value",
  token: "Token",
};

const TYPES = Object.keys(NAMES) as Type[];

const create = (
  type: library.IdentifierType,
  fields: library.Field[],
): library.Identifier => {
  switch (type) {
    case "can":
      return { type, id: 0, extended: false, fd: false };
    case "arinc429":
      return { type, label: 0, sdi: 0, sdiMatched: false };
    case "mil1553":
      return { type, rt: 0, subaddress: 0, direction: "receive", wordCount: 1 };
    case "field":
      return { type, field: fields[0]?.key ?? "", value: 0 };
    case "token":
      return { type, prefix: "" };
  }
};

const formatHex = (value: number): string => `0x${value.toString(16).toUpperCase()}`;

interface TypeFieldsProps {
  path: string;
  type: library.IdentifierType;
  fields: library.Field[];
}

const TypeFields = ({ path, type, fields }: TypeFieldsProps): ReactElement => {
  switch (type) {
    case "can":
      return (
        <>
          <Form.NumericField path={`${path}.id`} label="ID" />
          <OptionalField<number>
            path={`${path}.mask`}
            label="Mask"
            placeholder="Exact match"
            format={formatHex}
            parse={parseInteger}
          />
          <Form.SwitchField path={`${path}.extended`} label="Extended" />
          <Form.SwitchField path={`${path}.fd`} label="CAN FD" />
        </>
      );
    case "arinc429":
      return (
        <>
          <Form.NumericField path={`${path}.label`} label="Label" />
          <Form.NumericField path={`${path}.sdi`} label="SDI" />
          <Form.SwitchField path={`${path}.sdiMatched`} label="Match SDI" />
        </>
      );
    case "mil1553":
      return (
        <>
          <Form.NumericField path={`${path}.rt`} label="Remote terminal" />
          <Form.NumericField path={`${path}.subaddress`} label="Subaddress" />
          <Form.NumericField path={`${path}.wordCount`} label="Word count" />
          <Form.Field<library.Direction> path={`${path}.direction`} label="Direction">
            {(p) => (
              <Select.Buttons<library.Direction> {...p}>
                <Select.Item<library.Direction> itemKey="receive">Receive</Select.Item>
                <Select.Item<library.Direction> itemKey="transmit">
                  Transmit
                </Select.Item>
              </Select.Buttons>
            )}
          </Form.Field>
        </>
      );
    case "field":
      return (
        <>
          <Form.Field<string> path={`${path}.field`} label="Field">
            {(p) => (
              <Select.Simple<string> resourceName="field" {...p}>
                {fields.map(({ key, name }) => (
                  <Select.Item key={key} itemKey={key}>
                    {name}
                  </Select.Item>
                ))}
              </Select.Simple>
            )}
          </Form.Field>
          <Form.NumericField path={`${path}.value`} label="Value" />
        </>
      );
    case "token":
      return <Form.TextField path={`${path}.prefix`} label="Prefix" />;
  }
};

export interface IdentifierProps {
  /** The form path of the message entry. */
  path: string;
}

/** Edits how a message selects the frames that belong to it. */
export const Identifier = ({ path }: IdentifierProps): ReactElement => {
  const identifierPath = `${path}.identifier`;
  const { set } = Form.useContext();
  const identifier = Form.useFieldValue<library.Identifier>(identifierPath, {
    optional: true,
  });
  const fields = Form.useFieldValue<library.Field[]>(`${path}.fields`);
  const handleChange = (type: Type) =>
    set(identifierPath, type === "none" ? undefined : create(type, fields));
  return (
    <Flex.Box x wrap gap="large">
      <Input.Item label="Identifier" padHelpText>
        <Select.Simple<Type>
          resourceName="identifier"
          value={identifier?.type ?? "none"}
          onChange={handleChange}
        >
          {TYPES.map((type) => (
            <Select.Item key={type} itemKey={type}>
              {NAMES[type]}
            </Select.Item>
          ))}
        </Select.Simple>
      </Input.Item>
      {identifier != null && (
        <TypeFields path={identifierPath} type={identifier.type} fields={fields} />
      )}
    </Flex.Box>
  );
};
