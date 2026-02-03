// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";
import {
  type $ZodFunction,
  type $ZodFunctionIn,
  type $ZodFunctionOut,
} from "zod/v4/core";

import { deep } from "@/deep";
import { type record } from "@/record";

export const functionOutput = <
  In extends $ZodFunctionIn,
  Out extends $ZodFunctionOut,
  Func extends $ZodFunction<In, Out>,
>(
  schema: $ZodFunction<In, Out>,
): Func["_zod"]["def"]["output"] => schema._zod.def.output;

export const getFieldSchemaPath = (path: string): string =>
  deep.transformPath(path, (part, index, parts) => {
    const isLast = index === parts.length - 1;
    const isNumericPart = !isNaN(parseInt(part));
    const nextPartIsNumeric = !isNaN(parseInt(parts[index + 1]));
    if (isNumericPart) part = "element";
    if (isLast || nextPartIsNumeric) return part;
    return [part, "shape"];
  });

const sourceTypeGetter = (obj: unknown, key: string): z.ZodAny | null => {
  if (obj == null) return null;
  const v = (obj as Record<string, z.ZodAny>)[key];
  if (v == null && typeof obj === "object" && "sourceType" in obj) {
    const sourceType = (
      obj as { sourceType: () => z.ZodObject<z.ZodRawShape> }
    ).sourceType();
    return (sourceType as unknown as record.Unknown)[key] as z.ZodAny | null;
  }
  return v;
};

export const getFieldSchema: deep.TypedGet<z.ZodType, z.ZodType> = ((
  schema: z.ZodType,
  path: string,
  options?: Omit<deep.GetOptions, "getter">,
): z.ZodType | null => {
  if (path === "") return schema;
  return deep.get<z.ZodType, z.ZodType>(
    sourceTypeGetter(schema, "shape") as unknown as z.ZodObject<z.ZodRawShape>,
    getFieldSchemaPath(path),
    { ...options, getter: sourceTypeGetter } as deep.GetOptions<boolean | undefined>,
  );
}) as deep.TypedGet<z.ZodType, z.ZodType>;
