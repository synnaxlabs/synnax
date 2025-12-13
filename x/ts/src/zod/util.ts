// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { deep } from "@/deep";
import { type record } from "@/record";

/**
 * Gets the output schema from a z.function() schema.
 *
 * @example
 * functionOutput(z.function()) // z.ZodUnknown
 * functionOutput(z.function({ output: z.void() })) // z.ZodVoid
 * functionOutput(z.function({ output: z.number() })) // z.ZodNumber
 */
type FunctionOutput<T> = T extends z.ZodFunction<z.ZodTuple, infer O> ? O : z.ZodType;

export const functionOutput = <T extends z.ZodFunction>(
  schema: T,
): FunctionOutput<T> => {
  const def = schema._def as unknown as { output: FunctionOutput<T> };
  return def.output;
};

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
