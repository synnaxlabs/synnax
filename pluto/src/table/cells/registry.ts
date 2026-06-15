// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, table } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { type FC } from "react";
import { type z } from "zod";

import { type CellProps, Text, Value } from "@/table/cells/Cells";
import { type FormProps, TextForm, ValueForm } from "@/table/cells/Forms";
import { type Theming } from "@/theming";

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
  defaultConfig: (t: Theming.Theme) => ConfigOf<V>;
}

const value: Spec<"value"> = {
  key: "value",
  name: "Value",
  Form: ValueForm,
  Cell: Value,
  defaultConfig: (t) => ({
    variant: "value",
    channel: 0,
    rollingAverage: 1,
    precision: 2,
    notation: "standard",
    redline: { bounds: { lower: 0, upper: 1 }, gradient: [] },
    color: color.construct(t.colors.gray.l10),
    level: "h5",
    units: "",
    stalenessTimeout: 5,
    stalenessColor: t.colors.warning.m1,
  }),
  schema: table.CELL_CONFIG_SCHEMAS.value,
};

const text: Spec<"text"> = {
  key: "text",
  name: "Text",
  Form: TextForm,
  Cell: Text,
  defaultConfig: () => ({
    variant: "text",
    value: "",
    level: "h5",
    weight: 400,
    align: "center",
    backgroundColor: color.construct("#00000000"),
  }),
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
