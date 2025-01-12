// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { deep } from "@/deep";
import { type UnknownRecord } from "@/record";

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
    const sourceType = (obj as z.ZodEffects<z.ZodTypeAny>).sourceType();
    return (sourceType as unknown as UnknownRecord)[key] as z.ZodAny | null;
  }
  return v;
};

export const getFieldSchema: deep.TypedGet<z.ZodTypeAny, z.ZodTypeAny> = ((
  schema: z.ZodTypeAny,
  path: string,
  options?: Omit<deep.GetOptions, "getter">,
): z.ZodTypeAny | null =>
  deep.get<z.ZodTypeAny, z.ZodTypeAny>(
    sourceTypeGetter(schema, "shape") as unknown as z.AnyZodObject,
    getFieldSchemaPath(path),
    { ...options, getter: sourceTypeGetter } as deep.GetOptions<boolean | undefined>,
  ) as z.ZodTypeAny | null) as deep.TypedGet<z.ZodTypeAny, z.ZodTypeAny>;

export const bigInt = z.bigint().or(z.string().transform(BigInt));
