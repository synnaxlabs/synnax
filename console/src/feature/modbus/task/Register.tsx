// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import {
  READ_CHANNEL_SCHEMAS,
  type ReadChannel,
  type ReadChannelType,
} from "@/feature/modbus/task/types";
import { CSS } from "@/platform/css";

/** The name and description of each register type. */
export const REGISTERS: Record<ReadChannelType, { name: string; description: string }> =
  {
    coil: { name: "Coil", description: "Bit, read and write" },
    discrete_input: { name: "Discrete input", description: "Bit, read only" },
    input_register: { name: "Input register", description: "Word, read only" },
    holding_register: { name: "Holding register", description: "Word, read and write" },
    input_register_bit: {
      name: "Input register bit",
      description: "Bit of a word, read only",
    },
    holding_register_bit: {
      name: "Holding register bit",
      description: "Bit of a word, read and write",
    },
  };

export const REGISTER_TYPES = Object.keys(REGISTERS) as ReadChannelType[];

// A bit is one cell and a word two, and a bit of a word is one cell. The outline cut
// from the inside marks read only.
const SHAPES: Record<"bit" | "word", [outer: string, inner: string]> = {
  bit: ["M7.5 6h9v12h-9z", "M9.5 8v8h5V8z"],
  word: ["M2 6h9v12H2zM13 6h9v12h-9z", "M4 8v8h5V8zM15 8v8h5V8z"],
};

const WORD_TYPES = new Set<ReadChannelType>(["input_register", "holding_register"]);

const READ_ONLY_TYPES = new Set<ReadChannelType>([
  "discrete_input",
  "input_register",
  "input_register_bit",
]);

export interface RegisterGlyphProps {
  type: ReadChannelType;
}

/** A register type as its width in cells, solid when it can be written. */
export const RegisterGlyph = ({ type }: RegisterGlyphProps): ReactElement => {
  const [outer, inner] = SHAPES[WORD_TYPES.has(type) ? "word" : "bit"];
  const readOnly = READ_ONLY_TYPES.has(type);
  return (
    <svg
      viewBox="0 0 24 24"
      className={CSS.BE("modbus", "glyph")}
      fill="currentColor"
      aria-hidden
    >
      <path fillRule="evenodd" d={readOnly ? `${outer}${inner}` : outer} />
    </svg>
  );
};

/** Lets a menu grow to fit the descriptions beside the option names. */
export const MENU_PROPS = { className: CSS.BE("modbus", "menu") };

/** A quiet trigger that shows a picked option without its description. */
export const SHADOW_TRIGGER_PROPS: Select.SingleTriggerProps = {
  variant: "shadow",
  className: CSS.BE("modbus", "trigger"),
};

export interface OptionProps {
  glyph: ReactElement;
  name: string;
  description: string;
}

/** A menu option: a glyph, a name, and a description that the trigger hides. */
export const Option = ({ glyph, name, description }: OptionProps): ReactElement => (
  <>
    {glyph}
    <span>{name}</span>
    <span className={CSS.BE("modbus", "option-description")}>{description}</span>
  </>
);

export interface RegisterFieldProps {
  /** The channel's form path. */
  path: string;
}

/**
 * Selects a channel's register type. A new type keeps the fields that the type also
 * has, and resets the rest.
 */
export const RegisterField = ({ path }: RegisterFieldProps): ReactElement | null => {
  const field = Form.useField<ReadChannelType>(`${path}.type`, {
    onChange: (value, { get, set }) => {
      const prev = get<ReadChannel>(path).value;
      if (prev.type === value) return;
      const schema = READ_CHANNEL_SCHEMAS[value];
      const next = schema.parse({ type: value });
      set(path, { ...deep.overrideValidItems(next, prev, schema), type: value });
    },
  });
  if (field == null) return null;
  return (
    <Select.Simple<ReadChannelType>
      value={field.value}
      onChange={field.onChange}
      preview={field.preview}
      allowNone={false}
      resourceName="register type"
      triggerProps={SHADOW_TRIGGER_PROPS}
      dialogProps={MENU_PROPS}
    >
      {REGISTER_TYPES.map((t) => (
        <Select.Item<ReadChannelType>
          key={t}
          itemKey={t}
          aria-label={REGISTERS[t].name}
          aria-description={REGISTERS[t].description}
        >
          <Option glyph={<RegisterGlyph type={t} />} {...REGISTERS[t]} />
        </Select.Item>
      ))}
    </Select.Simple>
  );
};
