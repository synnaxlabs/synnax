// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { type FC } from "react";
import { z } from "zod";

import {
  type CellProps,
  Text,
  textPropsZ,
  Value,
  valuePropsZ,
} from "@/table/cells/Cells";
import { type FormProps, TextForm, ValueForm } from "@/table/cells/Forms";
import { telem } from "@/telem/aether";
import { type Theming } from "@/theming";
import { Value as CoreValue } from "@/vis/value";

const VARIANTS = ["text", "value"] as const;

export const variantZ = z.enum(VARIANTS);
export type Variant = z.infer<typeof variantZ>;

export interface Spec<Z extends z.ZodObject> {
  key: Variant;
  name: string;
  Form: FC<FormProps>;
  Cell: FC<CellProps<z.infer<Z>>>;
  schema: Z;
  defaultProps: (t: Theming.Theme) => z.infer<Z>;
}

const value: Spec<typeof valuePropsZ> = {
  key: "value",
  name: "Value",
  Form: ValueForm,
  Cell: Value,
  defaultProps: (t) => ({
    telem: telem.sourcePipeline("string", {
      connections: [
        { from: "valueStream", to: "rollingAverage" },
        { from: "rollingAverage", to: "stringifier" },
      ],
      segments: {
        valueStream: telem.streamChannelValue({ channel: 0 }),
        rollingAverage: telem.rollingAverage({ windowSize: 1 }),
        stringifier: telem.stringifyNumber({ precision: 2, notation: "standard" }),
      },
      outlet: "stringifier",
    }),
    redline: CoreValue.ZERO_READLINE,
    color: color.hex(t.colors.gray.l10),
    level: "h5",
    units: "",
    stalenessTimeout: 5,
    stalenessColor: t.colors.warning.m1,
  }),
  schema: valuePropsZ,
};

const text: Spec<typeof textPropsZ> = {
  key: "text",
  name: "Text",
  Form: TextForm,
  Cell: Text,
  defaultProps: () => ({
    value: "",
    level: "h5",
    units: "",
    weight: 400,
    align: "center",
    backgroundColor: "#00000000",
  }),
  schema: textPropsZ,
};

export const CELLS: Record<Variant, Spec<any>> = { text, value };
