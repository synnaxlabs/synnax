// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, table } from "@synnaxlabs/client";
import { type FC } from "react";
import { type z } from "zod";

import { type CellProps, Text, Value } from "@/table/cells/Cells";
import { type FormProps, TextForm, ValueForm } from "@/table/cells/Forms";

export const variantZ = table.cellConfigTypeZ;
export type Variant = table.CellConfigType;

export const configZ = table.cellConfigZ;
export type Config = table.CellConfig;
export type ConfigOf<V extends Variant> = Extract<Config, { variant: V }>;

export interface Spec<V extends Variant = Variant> {
  key: V;
  name: string;
  Form: FC<FormProps>;
  Cell: FC<CellProps<ConfigOf<V>>>;
  schema: z.ZodType<ConfigOf<V>>;
}

const value: Spec<"value"> = {
  key: "value",
  name: "Value",
  Form: ValueForm,
  Cell: Value,
  schema: table.CELL_CONFIG_SCHEMAS.value,
};

const text: Spec<"text"> = {
  key: "text",
  name: "Text",
  Form: TextForm,
  Cell: Text,
  schema: table.CELL_CONFIG_SCHEMAS.text,
};

export const REGISTRY = { text, value } as const satisfies {
  [V in Variant]: Spec<V>;
};

export const resolveSpec = (variant: string): Spec => {
  const spec = REGISTRY[variant as Variant];
  if (spec == null)
    throw new NotFoundError(`Table cell with variant ${variant} not found`);
  return spec as Spec;
};

/**
 * defaultConfig returns the default configuration for a cell variant, derived
 * by parsing the variant's schema with its declared field defaults. Fields
 * left absent by the schema (the theme-resolved text and staleness colors) are
 * filled in by the renderer at draw time.
 */
export const defaultConfig = <V extends Variant>(variant: V): ConfigOf<V> =>
  REGISTRY[variant].schema.parse({ variant }) as ConfigOf<V>;
