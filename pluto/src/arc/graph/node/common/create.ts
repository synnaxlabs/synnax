// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC } from "react";
import z from "zod/v4";

import { type Spec } from "@/arc/graph/node/types/spec";

export interface OperatorArgs<T extends string> {
  // key is the function type (e.g. "add", "gt").
  key: T;
  name: string;
  // Symbol renders the operator glyph on the canvas and in the palette. Operators carry
  // no parameters, so it ignores config.
  Symbol: FC;
}

// createOperator builds the spec for a parameter-less binary/unary operator: a glyph
// with fixed input/output handles, no config form. This is the operator analog of the
// schematic node factories in @/schematic/node/common/create.
export const createOperator = <T extends string>({
  key,
  name,
  Symbol,
}: OperatorArgs<T>) => {
  const configZ = z.object({ type: z.literal(key) });
  type Config = z.infer<typeof configZ>;
  const spec: Spec<T, Config> = {
    key,
    name,
    Form: () => null,
    Symbol,
    Preview: Symbol,
    defaultConfig: () => ({ type: key }),
    zIndex: 0,
  };
  return { configZ, spec };
};
