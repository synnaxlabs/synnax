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
import { Text } from "@synnaxlabs/lyra/text";
import { deep } from "@synnaxlabs/x";
import { type ReactElement, useId } from "react";

import {
  formatHex,
  FormattedField,
  parseInteger,
} from "@/feature/icd/editor/FormattedField";
import { CSS } from "@/platform/css";

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

interface FlagProps extends Pick<Form.UseFieldOptions<boolean>, "onChange"> {
  path: deep.Target<boolean>;
  label: string;
  disabled?: boolean;
}

const Flag = ({ path, label, disabled, onChange }: FlagProps): ReactElement => {
  const id = useId();
  const labelID = `${id}-label`;
  const field = Form.useField(path, { onChange });
  return (
    <Flex.Box x align="center" gap="small">
      <Input.Checkbox
        id={id}
        aria-labelledby={labelID}
        value={field.value}
        onChange={field.onChange}
        preview={field.preview}
        disabled={disabled}
      />
      <Text.Text
        el="label"
        id={labelID}
        htmlFor={id}
        color={disabled === true ? 8 : undefined}
      >
        {label}
      </Text.Text>
    </Flex.Box>
  );
};

interface CANFieldsProps {
  path: deep.Path<icd.CanIdentifier>;
}

const CANFields = ({ path }: CANFieldsProps): ReactElement => {
  const fd = Form.useFieldValue(path.fd);
  return (
    <>
      <Flex.Box x>
        <FormattedField<number>
          path={path.id}
          label="ID"
          format={formatHex}
          parse={parseInteger}
        />
        <FormattedField<number>
          path={path.mask}
          label="Mask"
          placeholder="Exact match"
          optional
          format={formatHex}
          parse={parseInteger}
        />
      </Flex.Box>
      <Input.Item label="Frame type" padHelpText>
        <Flex.Box x wrap align="center" className={CSS.BE("icd-editor", "flags")}>
          <Flag path={path.extended} label="Extended ID" />
          <Flag
            path={path.fd}
            label="CAN FD"
            onChange={(next, { set }) => {
              if (!next) set(path.bitrateSwitched, false);
            }}
          />
          <Flag path={path.bitrateSwitched} label="Bitrate switch" disabled={!fd} />
        </Flex.Box>
      </Input.Item>
    </>
  );
};

interface TypeFieldsProps {
  path: deep.Target<icd.Identifier | undefined>;
  type: icd.IdentifierType;
  fields: icd.BinaryField[];
}

const TypeFields = ({ path, type, fields }: TypeFieldsProps): ReactElement => {
  switch (type) {
    case "can":
      return <CANFields path={deep.variant(path, "type", "can")} />;
    case "field": {
      const field = deep.variant(path, "type", "field");
      return (
        <Flex.Box x>
          <Form.Field path={field.field} label="Field">
            {(p) => (
              <Select.Simple<string> resourceName="field" variant="floating" {...p}>
                {fields.map(({ key, name }) => (
                  <Select.Item key={key} itemKey={key}>
                    {name}
                  </Select.Item>
                ))}
              </Select.Simple>
            )}
          </Form.Field>
          <Form.NumericField path={field.value} label="Value" />
        </Flex.Box>
      );
    }
  }
};

export interface IdentifierProps {
  /** The form path of the binary payload. */
  path: deep.Path<icd.BinaryPayload>;
}

/** Picks how a binary message selects the frames that belong to it. */
export const Identifier = ({ path }: IdentifierProps): ReactElement => {
  const { set, mode } = Form.useContext();
  const type = Form.useFieldValue(deep.present(path.identifier).type, {
    optional: true,
  });
  const fields = Form.useFieldValue(path.fields);
  const handleChange = (next: Type) =>
    set(path.identifier, next === "none" ? undefined : create(next, fields));
  // A field identifier must name a field, so it needs one to pick.
  const types = TYPES.filter((t) => t !== "field" || fields.length > 0 || t === type);
  return (
    <Input.Item label="Identifier" padHelpText>
      <Select.Simple<Type>
        resourceName="identifier"
        value={type ?? "none"}
        onChange={handleChange}
        variant="floating"
        preview={mode === "preview"}
      >
        {types.map((t) => (
          <Select.Item key={t} itemKey={t}>
            {NAMES[t]}
          </Select.Item>
        ))}
      </Select.Simple>
    </Input.Item>
  );
};

/** Edits the values the message's identifier matches frames against. */
export const IdentifierFields = ({ path }: IdentifierProps): ReactElement | null => {
  const identifier = Form.useFieldValue(path.identifier, { optional: true });
  const fields = Form.useFieldValue(path.fields);
  if (identifier == null) return null;
  return <TypeFields path={path.identifier} type={identifier.type} fields={fields} />;
};
