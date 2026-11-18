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

type Type = icd.JSONMatchType | "none";

const NAMES: Record<Type, string> = {
  none: "Any document",
  string: "String",
  integer: "Signed integer",
  unsigned: "Unsigned integer",
  float: "Float",
  boolean: "Boolean",
};

const TYPES = Object.keys(NAMES) as Type[];

/** @returns a match of the type at the pointer, with the type's zero value. */
export const createMatch = (
  type: icd.JSONMatchType,
  pointer: string,
): icd.JSONMatch => {
  switch (type) {
    case "string":
      return { type, pointer, value: "" };
    case "integer":
      return { type, pointer, value: "0" };
    case "unsigned":
      return { type, pointer, value: "0" };
    case "float":
      return { type, pointer, value: 0 };
    case "boolean":
      return { type, pointer, value: true };
  }
};

const BOOLEANS = { false: false, true: true } as const;

type BooleanKey = keyof typeof BOOLEANS;

const POINTER_INPUT_PROPS = { placeholder: "/type" } as const;

interface ValueProps {
  path: deep.Target<icd.JSONMatch | undefined>;
  type: icd.JSONMatchType;
}

const Value = ({ path, type }: ValueProps): ReactElement => {
  switch (type) {
    case "float":
      return (
        <Form.NumericField
          path={deep.variant(path, "type", type).value}
          label="Value"
          padHelpText={false}
        />
      );
    case "integer":
    case "unsigned":
    case "string":
      return (
        <Form.TextField
          path={deep.variant(path, "type", type).value}
          label="Value"
          padHelpText={false}
        />
      );
    case "boolean":
      return (
        <Form.Field<boolean>
          path={deep.variant(path, "type", type).value}
          label="Value"
          padHelpText={false}
        >
          {({ value, onChange, preview }) => (
            <Select.Buttons<BooleanKey>
              value={String(value) as BooleanKey}
              onChange={(key: BooleanKey) => onChange(BOOLEANS[key])}
              preview={preview}
            >
              <Select.Item<BooleanKey> itemKey="false">False</Select.Item>
              <Select.Item<BooleanKey> itemKey="true">True</Select.Item>
            </Select.Buttons>
          )}
        </Form.Field>
      );
  }
};

export interface MatchProps {
  /** The form path of the JSON payload. */
  path: deep.Path<icd.JSONPayload>;
}

/**
 * Picks the JSON documents that belong to a message: the ones whose value at a
 * pointer equals a value. A type change keeps the pointer.
 */
export const Match = ({ path }: MatchProps): ReactElement => {
  const { get, set, mode } = Form.useContext();
  const match = deep.present(path.match);
  const type = Form.useFieldValue(match.type, { optional: true });
  const handleChange = (next: Type) => {
    if (next === type) return;
    if (next === "none") return set(path.match, undefined);
    const pointer = get(match.pointer, { optional: true })?.value ?? "";
    set(path.match, createMatch(next, pointer));
  };
  return (
    <>
      <Input.Item label="Match" padHelpText={false}>
        <Select.Simple<Type>
          resourceName="match"
          value={type ?? "none"}
          onChange={handleChange}
          variant="floating"
          allowNone={false}
          preview={mode === "preview"}
        >
          {TYPES.map((t) => (
            <Select.Item key={t} itemKey={t}>
              {NAMES[t]}
            </Select.Item>
          ))}
        </Select.Simple>
      </Input.Item>
      {type != null && (
        <Flex.Box x>
          <Form.TextField
            path={match.pointer}
            label="Pointer"
            padHelpText={false}
            inputProps={POINTER_INPUT_PROPS}
          />
          <Value path={path.match} type={type} />
        </Flex.Box>
      )}
    </>
  );
};
