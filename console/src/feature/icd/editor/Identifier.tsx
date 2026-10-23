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
import { type ReactElement } from "react";

import { OptionalField, parseInteger } from "@/feature/icd/editor/OptionalField";

type Type = icd.IdentifierType | "none";

const NAMES: Record<Type, string> = {
  none: "None",
  can: "CAN",
  field: "Field value",
};

const TYPES = Object.keys(NAMES) as Type[];

const create = (
  type: icd.IdentifierType,
  fields: icd.BinaryField[],
): icd.Identifier => {
  switch (type) {
    case "can":
      return { type, id: 0, extended: false, fd: false, bitrateSwitched: false };
    case "field":
      return { type, field: fields[0]?.key ?? "", value: 0 };
  }
};

const formatHex = (value: number): string => `0x${value.toString(16).toUpperCase()}`;

const CANFields = ({ path }: { path: string }): ReactElement => {
  const fd = Form.useFieldValue<boolean>(`${path}.fd`);
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
      <Form.SwitchField
        path={`${path}.fd`}
        label="CAN FD"
        onChange={(fd, { set }) => {
          if (!fd) set(`${path}.bitrateSwitched`, false);
        }}
      />
      {fd && (
        <Form.SwitchField path={`${path}.bitrateSwitched`} label="Bitrate switch" />
      )}
    </>
  );
};

interface TypeFieldsProps {
  path: string;
  type: icd.IdentifierType;
  fields: icd.BinaryField[];
}

const TypeFields = ({ path, type, fields }: TypeFieldsProps): ReactElement => {
  switch (type) {
    case "can":
      return <CANFields path={path} />;
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
  }
};

export interface IdentifierProps {
  /** The form path of the binary payload. */
  path: string;
}

/** Edits how a binary message selects the frames that belong to it. */
export const Identifier = ({ path }: IdentifierProps): ReactElement => {
  const identifierPath = `${path}.identifier`;
  const { set } = Form.useContext();
  const identifier = Form.useFieldValue<icd.Identifier>(identifierPath, {
    optional: true,
  });
  const fields = Form.useFieldValue<icd.BinaryField[]>(`${path}.fields`);
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
